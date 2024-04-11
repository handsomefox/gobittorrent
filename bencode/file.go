package bencode

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/jackpal/bencode-go"
)

type Torrent struct {
	File             File
	Info             Info
	AnnounceResponse AnnounceResponse
}

func (t *Torrent) DiscoverPeers() (AnnounceResponse, error) {
	return discoverPeers(t)
}

type File struct {
	Announce    string
	CreatedBy   string
	InfoHashSum [20]byte
	InfoHash    string
}

type Info struct {
	Name        string
	Pieces      []byte
	PieceHashes []string
	Length      int64
	PieceLength int64
}

func newTorrentFile(values any) (*Torrent, error) {
	torrent := new(Torrent)

	valuesMap, ok := values.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w, values (%q)", ErrConvertDecoded, values)
	}
	infoMap, ok := valuesMap["info"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w, values (%q)", ErrConvertDecoded, valuesMap)
	}
	announce, ok := valuesMap["announce"].(string)
	if !ok {
		return nil, ConvertError{ValueName: "announce", WantedType: "string"}
	}
	createdBy, ok := valuesMap["created by"].(string)
	if !ok {
		return nil, ConvertError{ValueName: "created by", WantedType: "string"}
	}
	length, ok := infoMap["length"].(int64)
	if !ok {
		return nil, ConvertError{ValueName: "length", WantedType: "int64"}
	}
	name, ok := infoMap["name"].(string)
	if !ok {
		return nil, ConvertError{ValueName: "name", WantedType: "string"}
	}
	pieceLength, ok := infoMap["piece length"].(int64)
	if !ok {
		return nil, ConvertError{ValueName: "piece length", WantedType: "int64"}
	}
	pieces, ok := infoMap["pieces"].(string)
	if !ok {
		return nil, ConvertError{ValueName: "pieces", WantedType: "string"}
	}

	// Marshal the info dictionary to get it's hash
	buffer := new(bytes.Buffer)
	// Using the library only for marshalling to get the info hash as i dont want to implement the whole marshalling process here
	if err := bencode.Marshal(buffer, infoMap); err != nil {
		return nil, fmt.Errorf("%w, because: %w", ErrBencodeInfoHash, err)
	}
	sum := sha1.Sum(buffer.Bytes())

	torrent.File = File{
		Announce:    announce,
		CreatedBy:   createdBy,
		InfoHashSum: sum,
		InfoHash:    hex.EncodeToString(sum[:]),
	}
	torrent.Info = Info{
		Length:      length,
		Name:        name,
		PieceLength: pieceLength,
		Pieces:      append(make([]byte, 0), pieces...),
		PieceHashes: make([]string, 0),
	}

	// Encode pieces
	start := 0
	for i := 1; i <= len(torrent.Info.Pieces); i++ {
		if i%20 == 0 {
			hashBytes := torrent.Info.Pieces[start:i]
			torrent.Info.PieceHashes = append(torrent.Info.PieceHashes, hex.EncodeToString(hashBytes))
			start = i
		}
	}

	return torrent, nil
}
