package bencode

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// DecodeValue returns the JSON representation of the decoded value.
func DecodeValue(encodedValue string) (string, error) {
	decoded, _, err := decodeValue(encodedValue)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(decoded)
	if err != nil {
		return "", MarshalError{Value: decoded, Message: err}
	}

	return string(b), nil
}

func DiscoverPeers(path string) (string, error) {
	torrent, err := parseTorrentFile(path)
	if err != nil {
		return "", err
	}

	resp, err := torrent.DiscoverPeers()
	if err != nil {
		return "", err
	}

	output := ""
	for _, peer := range resp.Peers {
		output += peer.IP.String() + ":" + strconv.FormatInt(int64(peer.Port), 10) + "\n"
	}

	return output, nil
}

func DecodeTorrentFile(path string) (string, error) {
	torrent, err := parseTorrentFile(path)
	if err != nil {
		return "", err
	}

	s := fmt.Sprintf("Tracker URL: %s\nLength: %d\nInfo Hash: %s\nPiece Length: %d\nPiece Hashes:\n",
		torrent.File.Announce,
		torrent.Info.Length,
		torrent.File.InfoHash,
		torrent.Info.PieceLength,
	)

	for i, h := range torrent.Info.PieceHashes {
		s += h
		if i != len(torrent.Info.PieceHashes)-1 {
			s += "\n"
		}
	}

	return s, nil
}

func RunHandshake(path string, addr string) (string, error) {
	torrent, err := parseTorrentFile(path)
	if err != nil {
		return "", err
	}

	resp, err := torrent.DiscoverPeers()
	if err != nil {
		return "", err
	}

	split := strings.Split(addr, ":")
	if len(split) != 2 {
		return "", ErrInvalidIPFormat
	}

	conn, id, err := tryHandshake(resp.Peers[0], torrent.File.InfoHashSum)
	if err != nil {
		return "", nil
	}
	defer conn.Close()

	return "Peer ID: " + id, nil
}

func parseTorrentFile(path string) (*Torrent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w %q, because: %w", ErrBencodeOpenFile, path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("%w %q, because: %w", ErrBencodeReadFile, path, err)
	}

	values, _, err := decodeValue(string(data))
	if err != nil {
		return nil, err
	}

	torrentFile, err := newTorrentFile(values)
	if err != nil {
		return nil, err
	}

	return torrentFile, err
}
