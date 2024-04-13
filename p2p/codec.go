package p2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type MessageID byte

const (
	Choke MessageID = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

type (
	Message struct {
		Length    uint32
		MessageID MessageID
		Payload   []byte
	}
	MessageEncoder  struct{ w io.Writer }
	MesssageDecoder struct{ r io.Reader }
)

func NewMessageEncoder(w io.Writer) *MessageEncoder  { return &MessageEncoder{w: w} }
func NewMessageDecoder(r io.Reader) *MesssageDecoder { return &MesssageDecoder{r: r} }

func (enc MessageEncoder) Encode(msg Message) error {
	buffer := new(bytes.Buffer)

	// Write the message length
	if err := binary.Write(buffer, binary.BigEndian, msg.Length); err != nil {
		return err
	}

	// Write the message type
	if err := buffer.WriteByte(byte(msg.MessageID)); err != nil {
		return err
	}

	// Write the payload
	if _, err := buffer.Write(msg.Payload); err != nil {
		return err
	}

	_, err := enc.w.Write(buffer.Bytes())

	return err
}

func (dec MesssageDecoder) Decode() (*Message, error) {
	buffer := new(bytes.Buffer)

	// First 4 bytes are the payload length
	_, err := io.CopyN(buffer, dec.r, 4)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(buffer.Bytes())

	buffer.Reset()
	// 1 byte for the message id
	_, err = io.CopyN(buffer, dec.r, 1)
	if err != nil {
		return nil, err
	}
	messageID := MessageID(buffer.Bytes()[0])

	buffer.Reset()
	// Everything else is the payload
	var payload []byte
	_, err = io.CopyN(buffer, dec.r, int64(length))
	if err != nil {
		payload = nil
	} else {
		payload = buffer.Bytes()
	}

	return &Message{
		Length:    length,
		MessageID: messageID,
		Payload:   payload,
	}, nil
}

type (
	HandshakeMessage struct {
		ProtocolLength uint8
		Protocol       string
		Reserved       [8]byte
		InfoHash       []byte
		PeerID         []byte
	}
	HandshakeEncoder struct{ w io.Writer }
	HandshakeDecoder struct{ r io.Reader }
)

func NewHandshakeEncoder(w io.Writer) *HandshakeEncoder { return &HandshakeEncoder{w: w} }
func NewHandshakeDecoder(r io.Reader) *HandshakeDecoder { return &HandshakeDecoder{r: r} }

func (enc *HandshakeEncoder) Encode(msg *HandshakeMessage) error {
	// 1. length of the protocol string (BitTorrent protocol) which is 19 (1 byte)
	_, err := enc.w.Write([]byte{19})
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 2. the string BitTorrent protocol (19 bytes)
	_, err = enc.w.Write([]byte("BitTorrent protocol"))
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 3. eight reserved bytes, which are all set to zero (8 bytes)
	_, err = enc.w.Write(make([]byte, 8))
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 4. sha1 infohash (20 bytes) (NOT the hexadecimal representation, which is 40 bytes long)
	_, err = enc.w.Write(msg.InfoHash)
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 5. peer id (20 bytes)
	_, err = enc.w.Write(msg.PeerID)
	if err != nil {
		return fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}

	return nil
}

func (dec *HandshakeDecoder) Decode() (*HandshakeMessage, error) {
	buf := make([]byte, 128)
	n, err := io.ReadAtLeast(dec.r, buf, HandshakeMessageLength)
	if err != nil {
		return nil, err
	}
	buf = buf[:n]

	if len(buf) < HandshakeMessageLength {
		return nil, fmt.Errorf("%w: received buffer is too small for a handshake message len=%d", ErrInvalidHandshakeFormat, len(buf))
	}

	msg := new(HandshakeMessage)

	index := 0

	msg.ProtocolLength = buf[index]
	if msg.ProtocolLength > HandshakeMessageLength-1 {
		return nil, fmt.Errorf("%w: protocol string length exceeds the message size", ErrInvalidHandshakeFormat)
	}
	index++

	msg.Protocol = string(buf[index:msg.ProtocolLength])
	index += int(msg.ProtocolLength)

	msg.Reserved = [8]byte{}
	index += 8

	msg.InfoHash = buf[index : index+20]
	index += 20

	msg.PeerID = buf[index : index+20]

	return msg, nil
}
