package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
)

const HandshakeMessageLength = 68

// Client is the strcture for receiving and sending messages between bittorrent clients.
type Client struct {
	log *slog.Logger

	peer     Peer // Only supports one peer
	infohash [20]byte

	// stored  after the handshake if it was successfull.
	peerID string
	conn   net.Conn
}

// NewClient returns a new client that immediately tries to initiate a handshake with the peer.
func NewClient(peer Peer, infohash [20]byte, log *slog.Logger) (*Client, error) {
	c := &Client{
		log:      log,
		peer:     peer,
		infohash: infohash,
		peerID:   "",
		conn:     nil,
	}

	if err := c.startHandshake(); err != nil {
		return nil, err
	}

	return c, nil
}

// Listen starts a goroutine with client.handleConnection for every (right now = only one) connection.
func (c *Client) Listen()        { go c.handleConnection(c.conn) }
func (c *Client) Close() error   { return c.conn.Close() }
func (c *Client) PeerID() string { return c.peerID }

func (c *Client) startHandshake() error {
	addr := c.peer.Addr()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.conn = conn

	if err := c.sendHandshake(); err != nil {
		c.log.Error("Failed to handshake", "err", err, "addr", addr)
		return err
	}

	return nil
}

func (c *Client) sendHandshake() error {
	msg := &HandshakeMessage{
		InfoHash: c.infohash[:],
		PeerID:   []byte("00112233445566778899"),
	}

	if err := NewHandshakeEncoder(c.conn).Encode(msg); err != nil {
		return err
	}

	decoded, err := NewHandshakeDecoder(c.conn).Decode()
	if err != nil {
		return err
	}

	c.peerID = hex.EncodeToString(decoded.PeerID)

	return nil
}

// handleConnection is the main loop for handling the message exchange between clients.
func (c *Client) handleConnection(conn net.Conn) {
	var lastMessage *Message
	for {
		if lastMessage != nil {
			if err := c.exchangeMessages(conn, lastMessage); err != nil {
				c.log.Error("Error during message exchange", "err", err)
			}
		}

		next, err := c.readNext(conn, 1024)
		if err != nil {
			c.log.Error("Error while reading the next message", "err", err)
			lastMessage = nil
			continue
		}
		lastMessage = next
		fmt.Printf("lastMessage: %v\n", lastMessage)
	}
}

// exchangeMessages acts accordingly to the receivedMessage MessageID.
func (c *Client) exchangeMessages(conn net.Conn, receivedMessage *Message) error {
	switch receivedMessage.MessageID {
	case Bitfield: // Send an Interested message
		msg := Message{Length: 2, MessageID: Interested, Payload: []byte{}}
		if err := NewMessageEncoder(conn).Encode(msg); err != nil {
			return err
		}
	case Unchoke:

	default:
		c.log.Debug("Unexpected message type", "type", receivedMessage.MessageID)
	}

	return nil
}

// readNext is a helper for reading the next message from the connection.
func (c *Client) readNext(conn net.Conn, bufferSize int) (*Message, error) {
	buffer := make([]byte, 0, bufferSize)

	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}

	buffer = buffer[:n]
	msg, err := NewMessageDecoder(bytes.NewReader(buffer)).Decode()
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
