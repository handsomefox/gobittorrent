package p2p

import (
	"bufio"
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	_ encoding.BinaryMarshaler   = (*Command)(nil)
	_ encoding.BinaryUnmarshaler = (*Command)(nil)
)

type MessageID byte

const (
	CommandChoke MessageID = iota
	CommandUnchoke
	CommandInterested
	CommandNotInterested
	CommandHave
	CommandBitfield
	CommandRequest
	CommandPiece
	CommandCancel
)

func (id MessageID) String() string {
	switch id {
	case CommandChoke:
		return "Choke"
	case CommandUnchoke:
		return "Unchoke"
	case CommandInterested:
		return "Interested"
	case CommandNotInterested:
		return "NotInterested"
	case CommandHave:
		return "Have"
	case CommandBitfield:
		return "Bitfield"
	case CommandRequest:
		return "Request"
	case CommandPiece:
		return "Piece"
	case CommandCancel:
		return "Cancel"
	default:
		return strconv.FormatUint(uint64(id), 10)
	}
}

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
	if err := NewCommandEncoder(bufio.NewWriter(buf)).Encode(cmd); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (cmd *Command) UnmarshalBinary(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	decoded, err := NewCommandDecoder(r).Decode()
	if err != nil {
		return err
	}
	*cmd = *decoded
	return nil
}

func (cmd Command) String() string {
	return fmt.Sprintf("length=%d messageID=%s payload=%v", cmd.Length, cmd.MessageID, cmd.Payload)
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
	c := new(Command)

	// First 4 bytes are the payload length
	lengthBuffer := make([]byte, 4)
	_, err := io.ReadFull(dec.r, lengthBuffer)
	if err != nil {
		return nil, err
	}
	c.Length = binary.BigEndian.Uint32(lengthBuffer) - 1

	idBuffer := make([]byte, 1)
	_, err = io.ReadFull(dec.r, idBuffer)
	if err != nil {
		return nil, err
	}
	c.MessageID = MessageID(idBuffer[0])

	// Everything else is the payloadBuf
	c.Payload = make([]byte, c.Length)
	if c.Length > 0 {
		_, err := io.ReadFull(dec.r, c.Payload)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
	}

	return c, nil
}
