package p2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/handsomefox/gobittorrent/bencode"
)

const (
	HandshakeMessageLength               = 68
	ReadDeadline           time.Duration = time.Second * 2
	WriteDeadline          time.Duration = time.Second * 2
)

type Connection struct {
	net.Conn
	quitch chan struct{}
	peerID string
	peer   bencode.Peer
}

func (c *Connection) PeerID() string {
	return c.peerID
}

func (c *Connection) Addr() string {
	return c.peer.Addr()
}

// Client is the structure for receiving and sending messages between bittorrent clients.
type Client struct {
	log    *slog.Logger
	t      *bencode.Torrent
	quitch chan struct{}
	peerID []byte

	connections      map[string]*Connection
	connectionsCount atomic.Int64
	mu               sync.RWMutex
}

// NewClient returns a new client that immediately tries to initiate a handshake with the peer.
func NewClient(log *slog.Logger, peerID []byte, torrent *bencode.Torrent) (*Client, error) {
	c := &Client{
		log:              log,
		t:                torrent,
		peerID:           peerID,
		connectionsCount: atomic.Int64{},
		connections:      map[string]*Connection{},
		mu:               sync.RWMutex{},
		quitch:           make(chan struct{}),
	}

	announce, err := c.Announce(context.TODO())
	if err != nil {
		return nil, err
	}

	// TODO: Maybe continue refetching?
	if len(announce.Peers) < 1 {
		return nil, ErrNoPeers
	}

	go c.addMissingConnections(announce)
	go c.refetchAnnounce(int64(announce.Interval))

	return c, nil
}

// Connections returns all the current connections as a slice.
func (c *Client) Connections() []*Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	slog.Debug("adding connections")

	conns := make([]*Connection, 0)
	for _, conn := range c.connections {
		conns = append(conns, conn)
	}

	slog.Debug("added connections")

	return conns
}

func (c *Client) ConnectionCount() int64 {
	return c.connectionsCount.Load()
}

func (c *Client) addConnection(conn *Connection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connections[conn.peer.Addr()] = conn
}

// removeConnection removes connections based on their peer.Addr().
func (c *Client) removeConnection(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.connections[addr]
	if !ok {
		return
	}

	conn.quitch <- struct{}{}
	delete(c.connections, addr)
}

// clearConnections removes all entries from the connections map.
func (c *Client) clearConnections() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, conn := range c.connections {
		slog.Debug("Closing connection", "addr", conn.Addr())
		conn.quitch <- struct{}{}
		close(conn.quitch)
		slog.Debug("Closed connection", "addr", conn.Addr())
	}

	clear(c.connections)
}

// HasConnection returns whether or not the current connection pool includes the provided connection (addr).
func (c *Client) HasConnection(addr string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.connections[addr]
	slog.Debug("has connection", "ok", ok, "addr", addr)
	return ok
}

// Close closes all of the clients connections and stops the refetch announce goroutine.
func (c *Client) Close() error {
	slog.Debug("closing the client")
	c.quitch <- struct{}{}
	close(c.quitch)
	c.clearConnections()
	return nil
}

// Announce sends the request to the tracker to get the latest announce message.
func (c *Client) Announce(ctx context.Context) (*bencode.AnnounceResponse, error) {
	announceReq := bencode.AnnounceMessage{
		Announce:   c.t.File.Announce,
		InfoHash:   bencode.String(hex.EncodeToString(c.t.File.InfoHashSum[:])),
		PeerID:     "00112233445566778899",
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       c.t.File.Info.Length,
		Compact:    1,
	}

	u, err := announceReq.URL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	decoded, err := bencode.NewDecoder(bytes.NewReader(body)).Decode()
	if err != nil {
		return nil, err
	}

	announce := new(bencode.AnnounceResponse)

	if err := announce.Unmarshal(decoded); err != nil {
		return nil, err
	}

	return announce, nil
}

// DiscoverPeers returns the peers from the announce message.
func (c *Client) DiscoverPeers(ctx context.Context) ([]bencode.Peer, error) {
	announce, err := c.Announce(ctx)
	if err != nil {
		return nil, err
	}
	return announce.Peers, nil
}

// startHandshake does the handshake with the peer (by calling sendHandshake) and adds the connection to the pool in the client.
func (c *Client) startHandshake(peer bencode.Peer, infoHash [20]byte, peerID []byte) error {
	addr := peer.Addr()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	if err := c.sendHandshake(&Connection{
		Conn:   conn,
		quitch: make(chan struct{}),
		peer:   peer,
	}, infoHash, peerID); err != nil {
		c.log.Error("Failed to handshake", "err", err, "addr", addr)
		return err
	}

	return nil
}

// sendHandshake encodes a new handshake message to the connection and decodes the response.
// If everything is successfull, the connection is then added to the pool.
func (c *Client) sendHandshake(conn *Connection, infoHash [20]byte, peerID []byte) error {
	msg := &HandshakeMessage{
		InfoHash: infoHash[:],
		PeerID:   peerID,
	}

	if err := NewHandshakeEncoder(conn).Encode(msg); err != nil {
		return err
	}

	decoded, err := NewHandshakeDecoder(conn).Decode()
	if err != nil {
		return err
	}

	conn.peerID = hex.EncodeToString(decoded.PeerID)

	c.addConnection(conn)
	go c.handleConnection(conn)

	return nil
}

// handleConnection is the main loop for handling the message exchange between clients.
func (c *Client) handleConnection(conn *Connection) {
	c.connectionsCount.Add(1)
	defer c.connectionsCount.Add(-1)
	defer conn.Close()
	var (
		lastMessage *Command
		buffer      = make([]byte, 4096)
	)
	for {
		select {
		case <-conn.quitch:
			return
		default:
			if lastMessage != nil {
				if err := c.exchangeMessages(conn, lastMessage); err != nil {
					c.log.Error("Error during message exchange", "err", err)
				}
			}

			next, err := c.readNext(conn, buffer)
			if err != nil {
				c.log.Error("Error while reading the next message", "err", err)
				lastMessage = nil

				if errors.Is(err, io.EOF) || errors.Is(err, os.ErrDeadlineExceeded) {
					slog.Debug("Connection removing itself")
					go c.removeConnection(conn.Addr())
					continue
				}

				continue
			}
			lastMessage = next
		}
	}
}

// exchangeMessages acts accordingly to the receivedMessage MessageID.
func (c *Client) exchangeMessages(conn *Connection, receivedMessage *Command) error {
	slog.Info("Received a message", "msg", receivedMessage, "peer", conn.peer.Addr())
	if err := conn.SetWriteDeadline(time.Now().Add(WriteDeadline)); err != nil {
		return err
	}
	switch receivedMessage.MessageID {
	case Bitfield: // Send an Interested message
		msg := &Command{Length: 2, MessageID: Interested, Payload: []byte{}}
		if err := NewCommandEncoder(conn).Encode(msg); err != nil {
			return err
		}
	case Unchoke:

	default:
		c.log.Debug("Unexpected message type", "type", receivedMessage.MessageID)
	}

	return nil
}

// readNext is a helper for reading the next message from the connection.
func (c *Client) readNext(conn net.Conn, buffer []byte) (*Command, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ReadDeadline)); err != nil {
		return nil, err
	}

	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	buffer = buffer[:n]

	r := bytes.NewReader(buffer)
	msg, err := NewCommandDecoder(r).Decode()
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// refetchAnnounce refetches announce every interval * time.Second
// closes the peers that no longer exist in the announce and adds the new ones to the pool.
func (c *Client) refetchAnnounce(interval int64) {
	var (
		tt        = time.NewTicker(time.Second * time.Duration(interval))
		ctx       = context.Background()
		errCount  = 0
		maxErrors = 10
		once      sync.Once
	)

	slog.Debug("Starting refetching announce", "interval", interval)
	defer slog.Debug("Closing refetch announce")

	for {
		select {
		case <-tt.C:
			if errCount > maxErrors { // Close the client.
				once.Do(func() {
					c.Close()
				})
				continue
			}

			announce, err := c.Announce(ctx)
			if err != nil {
				errCount++
				slog.Debug("Failed to refetch annouce", "err", err, "err_count", errCount)
				continue
			}

			go c.removeMissingConnections(announce)
			go c.addMissingConnections(announce)

		case <-c.quitch:
			return
		}
	}
}

func (c *Client) removeMissingConnections(announce *bencode.AnnounceResponse) {
	for _, peer := range announce.Peers {
		if c.HasConnection(peer.Addr()) {
			slog.Debug("Removing a peer that is now missing", "peer", peer.Addr())
			c.removeConnection(peer.Addr())
		}
	}
}

func (c *Client) addMissingConnections(announce *bencode.AnnounceResponse) {
	for _, peer := range announce.Peers {
		if !c.HasConnection(peer.Addr()) {
			slog.Debug("Adding a missing peer", "peer", peer.Addr())
			go func() {
				if err := c.startHandshake(peer, c.t.File.InfoHashSum, c.peerID); err != nil {
					slog.Error("Handshake error", "err", err, "peer", peer.Addr())
				}
			}()
		}
	}
}

// newUnchokePayload is a helper for creating a payload for the Unchoke message type.
func (c *Client) newUnchokePayload(index, begin, length uint32) []byte {
	buf := make([]byte, 0)
	binary.BigEndian.PutUint32(buf, index)  // the zero-based piece index
	binary.BigEndian.PutUint32(buf, begin)  // the zero-based byte offset within the piece
	binary.BigEndian.PutUint32(buf, length) // the length of the block in bytes
	return buf
}
