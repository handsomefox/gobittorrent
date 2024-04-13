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

	"github.com/handsomefox/gobittorrent/bencode"
)

const HandshakeMessageLength = 68

type Connection struct {
	net.Conn
	peerID string
}

func (c Connection) PeerID() string {
	return c.peerID
}

// Client is the strcture for receiving and sending messages between bittorrent clients.
type Client struct {
	log  *slog.Logger
	t    *bencode.Torrent
	conn Connection // stored  after the handshake if it was successfull.
}

// NewClient returns a new client that immediately tries to initiate a handshake with the peer.
func NewClient(log *slog.Logger, peerID []byte, torrent *bencode.Torrent) (*Client, error) {
	c := &Client{
		log: log,
		conn: Connection{
			peerID: "",
			Conn:   nil,
		},
		t: torrent,
	}

	peers, err := c.DiscoverPeers(context.TODO())
	if err != nil {
		return nil, err
	}

	if len(peers) < 1 {
		return nil, ErrNoPeers
	}

	if err := c.startHandshake(peers[0], torrent.File.InfoHashSum, peerID); err != nil {
		return nil, err
	}

	return c, nil
}

// Listen starts a goroutine with client.handleConnection for every (right now = only one) connection.
func (c *Client) Listen()        { go c.handleConnection(c.conn) }
func (c *Client) Close() error   { return c.conn.Close() }
func (c *Client) PeerID() string { return c.conn.PeerID() }

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

func (c *Client) DiscoverPeers(ctx context.Context) ([]bencode.Peer, error) {
	announce, err := c.Announce(ctx)
	if err != nil {
		return nil, err
	}
	return announce.Peers, nil
}

func (c *Client) startHandshake(peer bencode.Peer, infoHash [20]byte, peerID []byte) error {
	addr := peer.Addr()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.conn = Connection{Conn: conn}

	if err := c.sendHandshake(infoHash, peerID); err != nil {
		c.log.Error("Failed to handshake", "err", err, "addr", addr)
		return err
	}

	return nil
}

func (c *Client) sendHandshake(infoHash [20]byte, peerID []byte) error {
	msg := &HandshakeMessage{
		InfoHash: infoHash[:],
		PeerID:   peerID,
	}

	if err := NewHandshakeEncoder(c.conn).Encode(msg); err != nil {
		return err
	}

	decoded, err := NewHandshakeDecoder(c.conn).Decode()
	if err != nil {
		return err
	}

	c.conn.peerID = hex.EncodeToString(decoded.PeerID)

	// c.handleConnection(c.conn)

	return nil
}

// handleConnection is the main loop for handling the message exchange between clients.
func (c *Client) handleConnection(conn net.Conn) {
	var (
		lastMessage *Command
		buffer      = make([]byte, 4096)
	)
	for {
		if lastMessage != nil {
			if err := c.exchangeMessages(conn, lastMessage); err != nil {
				c.log.Error("Error during message exchange", "err", err)
			}
		}

		next, err := c.readNext(conn, buffer)
		if err != nil {
			c.log.Error("Error while reading the next message", "err", err)
			lastMessage = nil

			if errors.Is(err, io.EOF) {
				return
			}

			continue
		}
		lastMessage = next
	}
}

// exchangeMessages acts accordingly to the receivedMessage MessageID.
func (c *Client) exchangeMessages(conn net.Conn, receivedMessage *Command) error {
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

// newUnchokePayload is a helper for creating a payload for the Unchoke message type.
func (c *Client) newUnchokePayload(index, begin, length uint32) []byte {
	buf := make([]byte, 0)
	binary.BigEndian.PutUint32(buf, index)  // the zero-based piece index
	binary.BigEndian.PutUint32(buf, begin)  // the zero-based byte offset within the piece
	binary.BigEndian.PutUint32(buf, length) // the length of the block in bytes
	return buf
}
