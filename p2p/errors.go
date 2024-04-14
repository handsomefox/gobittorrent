package p2p

import "errors"

var (
	ErrInvalidHandshakeFormat = errors.New("p2p: invalid handshake format")
	ErrNoPeers                = errors.New("p2p: no peers")
	ErrWriteConn              = errors.New("p2p: failed to write to the connection")
	ErrPieceNotFound          = errors.New("p2p: downloaded piece was not found in the buffer") // Should technically never happen?
	ErrInvalidPieceHash       = errors.New("p2p: invalid downloaded piece hash")
	ErrNoCommand              = errors.New("p2p: command from the connection was nil")
)
