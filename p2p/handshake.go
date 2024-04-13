package p2p

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
)

var (
	_ encoding.BinaryMarshaler   = (*HandshakeMessage)(nil)
	_ encoding.BinaryUnmarshaler = (*HandshakeMessage)(nil)
)

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

func (msg *HandshakeMessage) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	if err := NewHandshakeEncoder(buf).Encode(msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (msg *HandshakeMessage) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	decoded, err := NewHandshakeDecoder(r).Decode()
	if err != nil {
		return err
	}
	*msg = *decoded
	return nil
}

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
