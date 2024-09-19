package p2p

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/handsomefox/gobittorrent/bencode"
)

const (
	ChunkSize = 16 * 1024

	HandshakeMessageLength               = 68
	ReadDeadline           time.Duration = time.Second * 10
	WriteDeadline          time.Duration = time.Second * 10
)

type Piece struct {
	Hash           string
	Chunks         []int
	TotalSize      int
	DownloadedSize int
	Index          uint32
}

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

// Client is the structure for receiving and sending commands between bittorrent clients.
type Client struct {
	log    *slog.Logger
	t      *bencode.Torrent
	quitch chan struct{}
	peerID []byte

	pieceQueue chan *Piece

	conns      map[string]*Connection // Addr - Conn
	connsMu    sync.RWMutex
	connsCount atomic.Int64

	pieces          map[string][]byte // Hash - Data
	piecesMu        sync.RWMutex
	piecesCompleted atomic.Int64
}

// NewClient returns a new client that immediately tries to initiate a handshake with the peer.
func NewClient(log *slog.Logger, peerID []byte, torrent *bencode.Torrent) (*Client, error) {
	c := &Client{
		log:        log,
		t:          torrent,
		quitch:     make(chan struct{}),
		peerID:     peerID,
		pieceQueue: make(chan *Piece, len(torrent.File.Info.Pieces)),
		conns:      make(map[string]*Connection),
		connsCount: atomic.Int64{},
		connsMu:    sync.RWMutex{},
		pieces:     make(map[string][]byte),
		piecesMu:   sync.RWMutex{},
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

// Download starts the download and blocks until the download finished or errors out.
func (c *Client) Download(w io.Writer) error {
	pieces := c.Pieces()

	go func() {
		for _, p := range pieces {
			p := p
			c.pieceQueue <- &p
		}
	}()

	for c.piecesCompleted.Load() != int64(len(pieces)) {
		runtime.Gosched()
	}

	c.piecesMu.Lock()
	defer c.piecesMu.Unlock()

	for _, hash := range c.t.File.Info.PieceHashes {
		piece, ok := c.pieces[hash]
		if !ok {
			return ErrPieceNotFound
		}

		sumBytes := sha1.Sum(piece)
		sumHex := hex.EncodeToString(sumBytes[:])

		if sumHex != hash {
			return ErrInvalidPieceHash
		}

		if _, err := w.Write(piece); err != nil {
			return err
		}
	}

	return nil
}

// Connections returns all the current connections as a slice.
func (c *Client) Connections() []*Connection {
	c.connsMu.RLock()
	defer c.connsMu.RUnlock()

	slog.Debug("adding connections")

	conns := make([]*Connection, 0)
	for _, conn := range c.conns {
		conns = append(conns, conn)
	}

	slog.Debug("added connections")

	return conns
}

func (c *Client) ConnectionCount() int64 {
	return c.connsCount.Load()
}

// HasConnection returns whether or not the current connection pool includes the provided connection (addr).
func (c *Client) HasConnection(addr string) bool {
	c.connsMu.RLock()
	defer c.connsMu.RUnlock()

	_, ok := c.conns[addr]
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

func (c *Client) PieceLengths() []int {
	var (
		info    = &c.t.File.Info
		total   = bencode.Integer(0)
		lengths = make([]int, 0, len(info.PieceHashes))
	)
	for range info.PieceHashes {
		if total+info.PieceLength < info.Length {
			lengths = append(lengths, int(info.PieceLength))
			total += info.PieceLength
		} else {
			l := info.Length - total
			lengths = append(lengths, int(l))
			total += l
		}
	}
	return lengths
}

func (c *Client) Pieces() []Piece {
	lengths := c.PieceLengths()
	pieces := make([]Piece, 0, len(lengths))

	for i, l := range lengths {
		chunks := make([]int, 0)
		total := 0
		for total != l {
			if total+ChunkSize < l {
				chunks = append(chunks, ChunkSize)
				total += ChunkSize
			} else {
				l := l - total
				chunks = append(chunks, l)
				total += l
			}
		}

		pieces = append(pieces, Piece{
			Index:     uint32(i),
			Chunks:    chunks,
			TotalSize: total,
			Hash:      c.t.File.Info.PieceHashes[i],
		})
	}

	return pieces
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

func (c *Client) addConnection(conn *Connection) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	c.conns[conn.peer.Addr()] = conn
}

// removeConnection removes connections based on their peer.Addr().
func (c *Client) removeConnection(addr string) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	conn, ok := c.conns[addr]
	if !ok {
		return
	}

	conn.quitch <- struct{}{}
	delete(c.conns, addr)
}

// clearConnections removes all entries from the connections map.
func (c *Client) clearConnections() {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	for _, conn := range c.conns {
		slog.Debug("Closing connection", "addr", conn.Addr())
		conn.quitch <- struct{}{}
		close(conn.quitch)
		slog.Debug("Closed connection", "addr", conn.Addr())
	}

	clear(c.conns)
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

// handleConnection is the main loop for handling the command exchange between clients.
func (c *Client) handleConnection(conn *Connection) {
	c.connsCount.Add(1)
	defer c.connsCount.Add(-1)
	defer conn.Close()

	for {
		select {
		case <-conn.quitch:
			return
		case piece := <-c.pieceQueue:
			if err := c.tryDownloadPiece(conn, piece); err != nil {
				if errors.Is(err, io.EOF) {
					c.log.Info("Reached EOF, closing this connection")
					go func() {
						c.removeConnection(conn.Addr())
					}()
					continue
				}

				c.log.Error("Error downloading a piece", "err", err)

				go func() {
					slog.Info("Restarting the connection")
					c.removeConnection(conn.Addr())
					if err := c.startHandshake(conn.peer, c.t.File.InfoHashSum, c.peerID); err != nil {
						slog.Error("Error restarting the connection", "err", err)
					}
					c.pieceQueue <- piece
				}()

				continue
			}
		}
	}
}

// tryDownloadPiece tries to download the piece from the peer.
// On success, the piece is written to the c.pieces.
func (c *Client) tryDownloadPiece(conn *Connection, piece *Piece) error {
	c.log.Debug("Connection starting to download a piece", "addr", conn.Addr(), "piece", piece)
	var lastCommand *Command
	for {
		if lastCommand != nil {
			if err := c.exchangeCommands(conn, lastCommand, piece); err != nil {
				return err
			}
		}
		if err := conn.SetReadDeadline(time.Now().Add(ReadDeadline)); err != nil {
			return err
		}
		next, err := c.readNext(conn)
		if err != nil {
			return err
		}
		lastCommand = next
	}
}

// exchangeCommands acts accordingly to the command MessageID.
func (c *Client) exchangeCommands(conn *Connection, command *Command, piece *Piece) error {
	if err := conn.SetWriteDeadline(time.Now().Add(WriteDeadline)); err != nil {
		return err
	}
	switch command.MessageID {
	case CommandBitfield: // Send an Interested command
		return c.writecommand(conn, &Command{Length: 2, MessageID: CommandInterested, Payload: []byte{}})
	case CommandUnchoke:
		go func() {
			offset := 0
			for _, blockSize := range piece.Chunks {
				payload := newUnchokePayload(piece.Index, uint32(offset), uint32(blockSize))
				if err := c.writecommand(conn, &Command{
					Length:    2 + uint32(len(payload)),
					MessageID: CommandRequest,
					Payload:   payload,
				}); err != nil {
					c.log.Error("Failed to send request command", "err", err)
					return
				}
				offset += blockSize
			}
		}()
	case CommandPiece:
		c.piecesMu.Lock()
		defer c.piecesMu.Unlock()

		if _, ok := c.pieces[piece.Hash]; !ok {
			c.pieces[piece.Hash] = make([]byte, piece.TotalSize)
		}

		pieceBuf := c.pieces[piece.Hash]

		buf := command.Payload
		index := binary.BigEndian.Uint32(buf)
		buf = buf[4:]
		begin := binary.BigEndian.Uint32(buf)
		buf = buf[4:]
		block := buf

		copy(pieceBuf[begin:], block)

		piece.DownloadedSize += len(block)
		if piece.DownloadedSize == piece.TotalSize {
			c.piecesCompleted.Add(1)
			slog.Info("PIECE COMPLETED!")

			return io.EOF
		}
		_ = index
	default:
		c.log.Debug("Unexpected command type", "type", command.MessageID)
	}

	return nil
}

func (c *Client) writecommand(w io.Writer, command *Command) error {
	c.log.Debug("Sending command", "command", command)
	return NewCommandEncoder(w).Encode(command)
}

// readNext is a helper for reading the next command from the connection.
func (c *Client) readNext(r io.Reader) (*Command, error) {
	command, err := NewCommandDecoder(r).Decode()
	if err != nil {
		return nil, err
	}

	if command == nil {
		return nil, ErrNoCommand
	}

	c.log.Debug("Received command", "type", command.MessageID.String())

	return command, nil
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
					c.log.Error("Handshake error", "err", err, "peer", peer.Addr())
				}
			}()
		}
	}
}

// newUnchokePayload is a helper for creating a payload for the Unchoke command type.
func newUnchokePayload(index, begin, length uint32) []byte {
	total := make([]byte, 12)
	buf := total
	binary.BigEndian.PutUint32(buf, index) // the zero-based piece index
	buf = buf[4:]
	binary.BigEndian.PutUint32(buf, begin) // the zero-based byte offset within the piece
	buf = buf[4:]
	binary.BigEndian.PutUint32(buf, length) // the length of the block in bytes
	return total
}
