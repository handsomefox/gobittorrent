package p2p

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func NewPeer(data []byte) (Peer, error) {
	var p Peer

	if len(data) < 6 {
		return p, fmt.Errorf("%w, invalid peer format, expected (size=%d), got (size=%d)", ErrParsePeer, 6, len(data))
	}

	p.IP = []byte{data[0], data[1], data[2], data[3]}
	p.Port = binary.BigEndian.Uint16([]byte{data[4], data[5]})

	return p, nil
}

func (p Peer) Addr() string {
	return p.IP.String() + ":" + strconv.FormatUint(uint64(p.Port), 10)
}

func (p Peer) Empty() bool {
	return len(p.IP) == 0 && p.Port == 0
}
