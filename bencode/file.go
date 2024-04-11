package bencode

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce  string
	CreatedBy string
	InfoHash  string
	Info      TorrentInfo
}

type TorrentInfo struct {
	Name        string
	Pieces      []byte
	PieceHashes []string
	Length      int64
	PieceLength int64
}

func NewTorrentFile(values any) (*TorrentFile, error) {
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

	torrentFile := &TorrentFile{
		Announce:  announce,
		CreatedBy: createdBy,
		Info: TorrentInfo{
			Length:      length,
			Name:        name,
			PieceLength: pieceLength,
			Pieces:      append(make([]byte, 0), pieces...),
			PieceHashes: make([]string, 0),
		},
		InfoHash: hex.EncodeToString(sum[:]),
	}

	// Encode pieces
	start := 0
	for i := 1; i <= len(torrentFile.Info.Pieces); i++ {
		if i%20 == 0 {
			hashBytes := torrentFile.Info.Pieces[start:i]
			torrentFile.Info.PieceHashes = append(torrentFile.Info.PieceHashes, hex.EncodeToString(hashBytes))
			start = i
		}
	}

	return torrentFile, nil
}

func DecodeTorrentFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%w %q, because: %w", ErrBencodeOpenFile, path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("%w %q, because: %w", ErrBencodeReadFile, path, err)
	}

	values, _, err := decodeValue(string(data))
	if err != nil {
		return "", err
	}

	torrentFile, err := NewTorrentFile(values)
	if err != nil {
		return "", err
	}

	s := fmt.Sprintf("Tracker URL: %s\nLength: %d\nInfo Hash: %s\nPiece Length: %d\nPiece Hashes:\n",
		torrentFile.Announce,
		torrentFile.Info.Length,
		torrentFile.InfoHash,
		torrentFile.Info.PieceLength,
	)

	for i, h := range torrentFile.Info.PieceHashes {
		s += h
		if i != len(torrentFile.Info.PieceHashes)-1 {
			s += "\n"
		}
	}

	return s, nil
}
