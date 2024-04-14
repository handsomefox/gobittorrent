# gobittorrent

A working single-file cli torrent client written in Go.

This is a command-line program that can:

- [x] Decode _and_ encode bencoded values
- [x] Decode torrent files
- [x] Decode announce messages
- [x] Show the available peers
- [x] Do the handshake with multiple peers
- [x] Exchange messages with multiple peers _(partially)_
- [x] Download (single-file) files from peers

## Build

Run `make build` to build the project. The binary is stored in the `bin` folder inside of it.

## Usage

```txt
gobittorrent

Commands:
  decode <string>
    decodes a bencoded string and outputs it as json
  peers <.torrent file>
    shows the available peers for the given .torrent file
  info <.torrent file>
    shows the decoded representation of the .torrent file
  handshake <.torrent file> <peer>
    does the handshake with the given peer, which is a string that looks like: "host:port"
  download <.torrent file> <output file>
    downloads a single-file torrent to the specified file
  help
    display this message

Usage:
  gobittorrent decode 5:hello
  gobittorrent decode d3:foo3:bar5:helloi52ee
  gobittorrent peers sample.torrent
  gobittorrent info sample.torrent
  gobittorrent handshake sample.torrent 1.1.1.1:1111
  gobittorrent download sample.torrent ./output.txt
```
