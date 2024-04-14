package commands

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

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

	client, err := p2p.NewClient(slog.Default(), []byte("00112233445566778899"), torrent)
	if err != nil {
		return "", err
	}
	defer client.Close()

	peers, err := client.DiscoverPeers(context.Background())
	if err != nil {
		return "", err
	}

	output := ""
	for _, peer := range peers {
		output += peer.Addr() + "\n"
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
		hex.EncodeToString(torrent.File.InfoHashSum[:]),
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

var ErrPeerNotFound = errors.New("commands: peer not found")

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

	client, err := p2p.NewClient(slog.Default(), []byte("00112233445566778899"), torrent)
	if err != nil {
		return "", err
	}
	defer client.Close()

	peers, err := client.DiscoverPeers(context.Background())
	if err != nil {
		return "", err
	}

	var peer bencode.Peer
	for _, p := range peers {
		if p.Addr() == addr {
			peer = p
		}
	}

	if peer.Empty() {
		return "", ErrPeerNotFound
	}

	for !client.HasConnection(peer.Addr()) {
		time.Sleep(1 * time.Second)
	}

	conns := client.Connections()

	id := "not found"
	for _, conn := range conns {
		if conn.Addr() == peer.Addr() {
			id = conn.PeerID()
		}
	}

	return "Peer ID: " + id, nil
}
