package bencode

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
)

func tryHandshake(peer Peer, infohash [20]byte) (net.Conn, string, error) {
	conn, err := net.Dial("tcp", peer.IP.String()+":"+strconv.FormatInt(int64(peer.Port), 10))
	if err != nil {
		return nil, "", err
	}

	id, err := sendHandshake(conn, infohash)
	if err != nil {
		return nil, "", err
	}

	return conn, id, nil
}

func sendHandshake(conn net.Conn, infohash [20]byte) (string, error) {
	// Write:
	// 1. length of the protocol string (BitTorrent protocol) which is 19 (1 byte)
	_, err := conn.Write([]byte{19})
	if err != nil {
		return "", fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 2. the string BitTorrent protocol (19 bytes)
	_, err = conn.Write([]byte("BitTorrent protocol"))
	if err != nil {
		return "", fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 3. eight reserved bytes, which are all set to zero (8 bytes)
	_, err = conn.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return "", fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 4. sha1 infohash (20 bytes) (NOT the hexadecimal representation, which is 40 bytes long)
	_, err = conn.Write(infohash[:])
	if err != nil {
		return "", fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}
	// 5. peer id (20 bytes)
	_, err = conn.Write([]byte("00112233445566778899"))
	if err != nil {
		return "", fmt.Errorf("%w, because: %w", ErrWriteConn, err)
	}

	// Receive the handshake
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}
	buffer = buffer[:n]

	if len(buffer) != 68 {
		return "", ErrInvalidHandshakeResponse
	}

	return hex.EncodeToString(buffer[48:]), nil
}
