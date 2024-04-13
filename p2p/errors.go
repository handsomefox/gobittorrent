package p2p

import "errors"

var (
	ErrInvalidHandshakeResponse = errors.New("p2p: invalid handshake response")
	ErrParsePeer                = errors.New("p2p: failed to parse peer")
	ErrWriteConn                = errors.New("p2p: failed to write to the connection")
)
