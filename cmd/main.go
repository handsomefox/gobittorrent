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

const Usage = `gobittorrent

Commands:
  decode <string>
    decodes a bencoded string and outputs it as json
  peers <.torrent file>
    shows the available peers for the given .torrent file
  info <.torrent file>
    shows the decoded representation of the .torrent file
  handshake <.torrent file> <peer>
    does the handshake with the given peer, which is a string that looks like: "host:port"
  help
    display this message

Usage:
  gobittorrent decode 5:hello
  gobittorrent decode d3:foo3:bar5:helloi52ee
  gobittorrent peers sample.torrent
  gobittorrent info sample.torrent
  gobittorrent handshake sample.torrent 1.1.1.1:1111`

const IncorrectUsage = "Incorrect usage...\n" + Usage
