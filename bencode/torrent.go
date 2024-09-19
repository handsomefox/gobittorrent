package bencode

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
)

// Torrent is a structure that describes the .torrent file and related actions to it.
type Torrent struct {
	File File
}

// File is the contents of the file itself.
type File struct {
	Announce    String
	CreatedBy   String
	Info        Info
	InfoHashSum [20]byte
}

type Info struct {
	Name        String
	Pieces      []byte
	PieceHashes []string
	Length      Integer
	PieceLength Integer
}

func NewTorrent(r io.Reader) (*Torrent, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	decoded, err := NewDecoder(bytes.NewReader(data)).Decode()
	if err != nil {
		return nil, err
	}

	torrent, err := decodeTorrent(decoded)
	if err != nil {
		return nil, err
	}

	return torrent, err
}

func decodeTorrent(values any) (*Torrent, error) {
	torrent := new(Torrent)

	valuesMap, ok := values.(Dictionary)
	if !ok {
		return nil, fmt.Errorf("%w, values (%q)", ErrConvertDecoded, values)
	}
	infoMap, ok := valuesMap["info"].(Dictionary)
	if !ok {
		return nil, fmt.Errorf("%w, values (%q)", ErrConvertDecoded, valuesMap)
	}
	announce, ok := valuesMap["announce"].(String)
	if !ok {
		return nil, ConvertError{ValueName: "announce", WantedType: "String"}
	}
	createdBy, ok := valuesMap["created by"].(String)
	if !ok {
		return nil, ConvertError{ValueName: "created by", WantedType: "String"}
	}
	length, ok := infoMap["length"].(Integer)
	if !ok {
		return nil, ConvertError{ValueName: "length", WantedType: "Integer"}
	}
	name, ok := infoMap["name"].(String)
	if !ok {
		return nil, ConvertError{ValueName: "name", WantedType: "String"}
	}
	pieceLength, ok := infoMap["piece length"].(Integer)
	if !ok {
		return nil, ConvertError{ValueName: "piece length", WantedType: "Integer"}
	}
	pieces, ok := infoMap["pieces"].(String)
	if !ok {
		return nil, ConvertError{ValueName: "pieces", WantedType: "String"}
	}

	encoded, err := infoMap.Encode()
	if err != nil {
		return nil, fmt.Errorf("%w, because: %w", ErrBencodeInfoHash, err)
	}

	torrent.File = File{
		Announce:    announce,
		CreatedBy:   createdBy,
		InfoHashSum: sha1.Sum(encoded),
		Info: Info{
			Length:      length,
			Name:        name,
			PieceLength: pieceLength,
			Pieces:      append(make([]byte, 0), pieces...),
			PieceHashes: make([]string, 0),
		},
	}

	// Encode pieces
	start := 0
	for i := 1; i <= len(torrent.File.Info.Pieces); i++ {
		if i%20 == 0 {
			hashBytes := torrent.File.Info.Pieces[start:i]
			torrent.File.Info.PieceHashes = append(torrent.File.Info.PieceHashes, hex.EncodeToString(hashBytes))
			start = i
		}
	}

	return torrent, nil
}
