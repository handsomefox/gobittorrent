package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/handsomefox/gobittorrent/bencode"
	"github.com/handsomefox/gobittorrent/p2p"
)

type (
	CommandFunc  func(string) (string, error)
	CommandFunc2 func(string, string) (string, error)
)

func RunCommand(f CommandFunc) {
	if len(os.Args) < 2 {
		slog.Error("Invalid argument count", "want", 2, "got", len(os.Args))
		return
	}

	r, err := f(os.Args[2])
	if err != nil {
		slog.Error("Failed to run the command", "err", err)
	} else {
		fmt.Println(r)
	}
}

func RunCommand2(f CommandFunc2) {
	if len(os.Args) < 3 {
		slog.Error("Invalid argument count", "want", 3, "got", len(os.Args))
		return
	}

	r, err := f(os.Args[2], os.Args[3])
	if err != nil {
		slog.Error("Failed to run the command", "err", err)
	} else {
		fmt.Println(r)
	}
}

// Decode returns the JSON representation of the decoded value.
func Decode(encodedValue string) (string, error) {
	decoded, err := bencode.NewDecoder(strings.NewReader((encodedValue))).Decode()
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func Peers(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	torrent, err := bencode.NewTorrent(f)
	if err != nil {
		return "", err
	}

	resp, err := torrent.DiscoverPeers(context.Background())
	if err != nil {
		return "", err
	}

	output := ""
	for _, peer := range resp.Peers {
		output += peer.IP.String() + ":" + strconv.FormatInt(int64(peer.Port), 10) + "\n"
	}

	return output, nil
}

func Info(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	torrent, err := bencode.NewTorrent(f)
	if err != nil {
		return "", err
	}

	s := fmt.Sprintf("Tracker URL: %s\nLength: %d\nInfo Hash: %s\nPiece Length: %d\nPiece Hashes:\n",
		torrent.File.Announce,
		torrent.File.Info.Length,
		torrent.File.InfoHash,
		torrent.File.Info.PieceLength,
	)

	for i, h := range torrent.File.Info.PieceHashes {
		s += h
		if i != len(torrent.File.Info.PieceHashes)-1 {
			s += "\n"
		}
	}

	return s, nil
}

func Handshake(path, addr string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	torrent, err := bencode.NewTorrent(f)
	if err != nil {
		return "", err
	}

	resp, err := torrent.DiscoverPeers(context.Background())
	if err != nil {
		return "", err
	}

	var peer p2p.Peer
	for _, p := range resp.Peers {
		if p.Addr() == addr {
			peer = p
		}
	}

	if peer.Empty() {
		return "", fmt.Errorf("failed to find the correct peer")
	}

	client, err := p2p.NewClient(peer, torrent.File.InfoHashSum, slog.Default())
	if err != nil {
		return "", err
	}
	defer client.Close()

	return "Peer ID: " + client.PeerID(), nil
}
