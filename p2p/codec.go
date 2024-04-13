package p2p

import (
	"bytes"
	"encoding/binary"
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

type Message struct {
	Length    uint32
	MessageID MessageID
	Payload   []byte
}

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

func (enc Encoder) Encode(msg Message) error {
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

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

func (dec Decoder) Decode() (*Message, error) {
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
