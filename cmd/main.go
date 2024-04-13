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

	if len(os.Args) < 2 {
		fmt.Println(commands.IncorrectUsageMessageString())
	}

	switch command := strings.ToLower(os.Args[1]); command {
	case "decode":
		decoded, err := commands.Decode(os.Args[2])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(decoded)
		}
	case "peers":
		peers, err := commands.Peers(os.Args[2])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(peers)
		}
	case "help":
		fmt.Println(commands.UsageMessageString())
	case "info":
		decoded, err := commands.Info(os.Args[2])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(decoded)
		}
	case "handshake":
		if len(os.Args) < 3 {
			fmt.Println(commands.IncorrectUsageMessageString())
			return
		}
		out, err := commands.Handshake(os.Args[2], os.Args[3])
		if err != nil {
			slog.Error(err.Error())
		} else {
			fmt.Println(out)
		}
	default:
		fmt.Println(commands.UsageMessageString())
	}
}
