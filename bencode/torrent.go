package bencode

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/jackpal/bencode-go"
)

// Torrent is a structure that describes the .torrent file and related actions to it.
type Torrent struct {
	AnnounceResponse AnnounceResponse
	File             File
}

// File is the contents of the file itself.
type File struct {
	Announce    string
	CreatedBy   string
	InfoHash    string
	Info        Info
	InfoHashSum [20]byte
}

type Info struct {
	Name        string
	Pieces      []byte
	PieceHashes []string
	Length      int64
	PieceLength int64
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

	torrentFile, err := decodeTorrent(decoded)
	if err != nil {
		return nil, err
	}

	return torrentFile, err
}

func (t *Torrent) DiscoverPeers(ctx context.Context) (AnnounceResponse, error) {
	announceReq := AnnounceRequest{
		Announce:   t.File.Announce,
		InfoHash:   t.File.InfoHash,
		PeerID:     "00112233445566778899",
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       t.File.Info.Length,
		Compact:    1,
	}

	u, err := announceReq.Encode()
	if err != nil {
		return AnnounceResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return AnnounceResponse{}, fmt.Errorf("%w %q, because: %w", ErrGetAnnounce, u, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AnnounceResponse{}, fmt.Errorf("%w %q, because: %w", ErrGetAnnounce, u, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AnnounceResponse{}, fmt.Errorf("%w, because: %w", ErrDecodeAnnounceBody, err)
	}

	decoded, err := NewDecoder(bytes.NewReader(body)).Decode()
	if err != nil {
		return AnnounceResponse{}, err
	}

	announcementResponse, err := DecodeAnnounceResponse(decoded)
	if err != nil {
		return AnnounceResponse{}, err
	}

	return announcementResponse, nil
}

func decodeTorrent(values any) (*Torrent, error) {
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
