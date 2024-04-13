package p2p

import "errors"

var (
	ErrInvalidHandshakeFormat = errors.New("p2p: invalid handshake format")
	ErrParsePeer              = errors.New("p2p: failed to parse peer")
	ErrWriteConn              = errors.New("p2p: failed to write to the connection")
)
