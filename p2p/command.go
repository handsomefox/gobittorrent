package p2p

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"io"
)

var (
	_ encoding.BinaryMarshaler   = (*Command)(nil)
	_ encoding.BinaryUnmarshaler = (*Command)(nil)
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
	Command struct {
		Length    uint32
		MessageID MessageID
		Payload   []byte
	}
	CommandEncoder struct{ w io.Writer }
	CommandDecoder struct{ r io.Reader }
)

func NewCommandEncoder(w io.Writer) *CommandEncoder { return &CommandEncoder{w: w} }
func NewCommandDecoder(r io.Reader) *CommandDecoder { return &CommandDecoder{r: r} }

func (cmd *Command) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	if err := NewCommandEncoder(buf).Encode(cmd); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (cmd *Command) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	decoded, err := NewCommandDecoder(r).Decode()
	if err != nil {
		return err
	}
	*cmd = *decoded
	return nil
}

func (enc CommandEncoder) Encode(cmd *Command) error {
	buffer := new(bytes.Buffer)

	// Write the command length
	if err := binary.Write(buffer, binary.BigEndian, cmd.Length); err != nil {
		return err
	}

	// Write the command type
	if err := buffer.WriteByte(byte(cmd.MessageID)); err != nil {
		return err
	}

	// Write the payload
	if _, err := buffer.Write(cmd.Payload); err != nil {
		return err
	}

	_, err := enc.w.Write(buffer.Bytes())

	return err
}

func (dec CommandDecoder) Decode() (*Command, error) {
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

	return &Command{
		Length:    length,
		MessageID: messageID,
		Payload:   payload,
	}, nil
}
