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

test: vet
	go test -v -race ./...
.PHONY: test

build: lint test
	go build -ldflags "-s -w" -o ./bin/gobittorrent cmd/main.go
.PHONY:build

help: build
	./bin/gobittorrent help
.PHONY:help

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
	./bin/gobittorrent handshake sample.torrent 165.232.33.77:51467
.PHONY:handshake

run_handshake: lint test
	go run -race ./cmd/main.go handshake sample.torrent 165.232.33.77:51467
.PHONY:run_handshake
