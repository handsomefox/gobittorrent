.DEFAULT_GOAL := build

fmt:
	gofumpt -l -w .
.PHONY:fmt

lint: fmt
	golangci-lint run --issues-exit-code 0 ./...
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: lint vet
	go build -ldflags "-s -w" -o ./bin/gobittorrent cmd/main.go
.PHONY:build

decode: build
	./bin/gobittorrent decode d3:foo3:bar5:helloi52ee
.PHONY:decode

peers: build
	./bin/gobittorrent peers sample.torrent
.PHONY:peers

info: build
	./bin/gobittorrent info sample.torrent
.PHONY:info

handshake: build
	./bin/gobittorrent handshake sample.torrent 178.62.82.89:51448
.PHONY:handshake
