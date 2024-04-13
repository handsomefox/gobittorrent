package p2p

import "errors"

var (
	ErrInvalidHandshakeFormat = errors.New("p2p: invalid handshake format")
	ErrNoPeers                = errors.New("p2p: no peers")
	ErrWriteConn              = errors.New("p2p: failed to write to the connection")
)
