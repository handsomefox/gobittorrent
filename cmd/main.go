package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/handsomefox/gobittorrent/cmd/commands"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
	slog.SetDefault(log)
	switch command := strings.ToLower(os.Args[1]); command {
	case "decode":
		commands.RunCommand(commands.Decode)
	case "peers":
		commands.RunCommand(commands.Peers)
	case "help":
		fmt.Println(Usage)
	case "info":
		commands.RunCommand(commands.Info)
	case "handshake":
		commands.RunCommand2(commands.Handshake)
	default:
		fmt.Println(IncorrectUsage)
	}
}

const Usage = `Usage:
gobittorrent decode 5:hello
gobittorrent decode d3:foo3:bar5:helloi52ee
gobittorrent peers sample.torrent
gobittorrent info sample.torrent
gobittorrent handshake sample.torrent 1.1.1.1:1111

To display this message use:
gobittorrent help`

const IncorrectUsage = "Incorrect usage...\n" + Usage
