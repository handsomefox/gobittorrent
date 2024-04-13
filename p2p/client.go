package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strconv"
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

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) PeerID() string {
	return c.peerID
}

func (c *Client) startHandshake() error {
	conn, err := net.Dial("tcp", c.peer.IP.String()+":"+strconv.FormatInt(int64(c.peer.Port), 10))
	if err != nil {
		return err
	}

	c.conn = conn

	if err := c.sendHandshake(); err != nil {
		return err
	}

	return nil
}

func (c *Client) sendHandshake() error {
	// Write:
	// 1. length of the protocol string (BitTorrent protocol) which is 19 (1 byte)
	_, err := c.conn.Write([]byte{19})
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 2. the string BitTorrent protocol (19 bytes)
	_, err = c.conn.Write([]byte("BitTorrent protocol"))
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 3. eight reserved bytes, which are all set to zero (8 bytes)
	_, err = c.conn.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 4. sha1 infohash (20 bytes) (NOT the hexadecimal representation, which is 40 bytes long)
	_, err = c.conn.Write(c.infohash[:])
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 5. peer id (20 bytes)
	_, err = c.conn.Write([]byte("00112233445566778899"))
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}

	// Receive the handshake
	buffer := make([]byte, 0, 128)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return err
	}
	buffer = buffer[:n]

	if len(buffer) != HandshakeMessageLength {
		return ErrInvalidHandshakeResponse
	}

	c.peerID = hex.EncodeToString(buffer[48:])

	return nil
}

// Listen starts a goroutine with client.handleConnection for every (right now = only one) connection.
func (c *Client) Listen() {
	go c.handleConnection(c.conn)
}

// handleConnection is the main loop for handling the message exchange between clients.
func (c *Client) handleConnection(conn net.Conn) {
	var lastMessage *Message
	for {
		if lastMessage != nil {
			if err := c.exchangeMessages(conn, lastMessage); err != nil {
				c.log.Error("error during message exchange", "err", err)
			}
		}

		next, err := c.readNext(conn, 1024)
		if err != nil {
			c.log.Error("error while reading the next message", "err", err)
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
		if err := NewEncoder(conn).Encode(msg); err != nil {
			return err
		}
	case Unchoke:
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
	msg, err := NewDecoder(bytes.NewReader(buffer)).Decode()
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
