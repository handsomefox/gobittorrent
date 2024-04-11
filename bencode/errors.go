package bencode

import (
	"errors"
	"fmt"
)

var (
	ErrGetAnnounce              = errors.New("bencode: failed to GET the announce")
	ErrDecodeAnnounceBody       = errors.New("bencode: failed to decode the announce body")
	ErrParseAnnounceURL         = errors.New("bencode: failed to parse the announce url")
	ErrParsePeer                = errors.New("bencode: failed to parse peer")
	ErrConvertDecoded           = errors.New("bencode: failed to convert decoded values to a map")
	ErrBencodeInfoHash          = errors.New("bencode: failed to bencode info_hash")
	ErrBencodeOpenFile          = errors.New("bencode: failed to open the file")
	ErrBencodeReadFile          = errors.New("bencode: failed to read the file")
	ErrMarshal                  = errors.New("bencode: failed to marshal a value")
	ErrUnknownValueType         = errors.New("bencode: unknown value type")
	ErrWriteConn                = errors.New("bencode: failed to write to the connection")
	ErrInvalidIPFormat          = errors.New("bencode: invalid IP format provided, please use <ip>:<port>")
	ErrInvalidHandshakeResponse = errors.New("bencode: invalid handshake response")
)

type ConvertError struct {
	ValueName  string
	WantedType string
}

func (err ConvertError) Error() string {
	return fmt.Sprintf("bencode: failed to convert the field %q to the wanted type %q", err.ValueName, err.WantedType)
}

type MarshalError struct {
	Message error
	Value   any
}

func (err MarshalError) Error() string {
	return fmt.Errorf("bencode: failed to marshal the value %q, because: %w", err.Value, err.Message).Error()
}

type SyntaxError struct {
	Message string
}

func NewSyntaxError(message string) SyntaxError {
	return SyntaxError{
		Message: message,
	}
}

func NewSyntaxErrorf(message string, args ...any) SyntaxError {
	return SyntaxError{
		Message: fmt.Sprintf(message, args...),
	}
}

func (err SyntaxError) Error() string {
	return err.Message
}
