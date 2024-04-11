package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/handsomefox/gobittorrent/bencode"
)

func init() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
	slog.SetDefault(log)
}

func main() {
	if len(os.Args) < 2 {
		PrintIncorrectUsage()
	}

	switch command := strings.ToLower(os.Args[1]); command {
	case "decode":
		decoded, err := bencode.DecodeValue(os.Args[2])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(decoded)
		}
	case "peers":
		peers, err := bencode.DiscoverPeers(os.Args[2])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(peers)
		}
	case "help":
		PrintUsage()
	case "info":
		decoded, err := bencode.DecodeTorrentFile(os.Args[2])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(decoded)
		}
	case "handshake":
		if len(os.Args) < 3 {
			PrintIncorrectUsage()
			return
		}
		out, err := bencode.RunHandshake(os.Args[2], os.Args[3])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(out)
		}
	default:
		PrintUsage()
	}
}

func PrintIncorrectUsage() {
	fmt.Println("Incorrect usage.")
	PrintUsage()
}

func PrintUsage() {
	const usage = `Usage:
	gobittorrent decode 5:hello
	`
	fmt.Println(usage)
}
