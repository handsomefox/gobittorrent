package bencode

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
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

func discoverPeers(torrent *Torrent) (AnnounceResponse, error) {
	req := AnnounceRequest{
		Announce:   torrent.File.Announce,
		InfoHash:   torrent.File.InfoHash,
		PeerID:     "00112233445566778899",
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       torrent.Info.Length,
		Compact:    1,
	}

	u, err := req.Encode()
	if err != nil {
		return AnnounceResponse{}, err
	}

	resp, err := http.Get(u)
	if err != nil {
		return AnnounceResponse{}, fmt.Errorf("%w %q, because: %w", ErrGetAnnounce, u, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AnnounceResponse{}, fmt.Errorf("%w, because: %w", ErrDecodeAnnounceBody, err)
	}

	decoded, _, err := decodeValue(string(body))
	if err != nil {
		return AnnounceResponse{}, err
	}

	announcementResponse, err := DecodeAnnounceResponse(decoded)
	if err != nil {
		return AnnounceResponse{}, err
	}

	return announcementResponse, nil
}
