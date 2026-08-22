.PHONY: build uci test vet run up dev bench

# This project has no cgo dependencies, and the Docker build already sets this.
# Keeping it off locally also avoids a Go 1.22 / recent-macOS link failure
# ("missing LC_UUID load command") in test binaries that pull in net.
export CGO_ENABLED = 0

# Build the HTTP/WebSocket game server.
build:
	go build -o bin/server .

# Build the UCI engine binary (point a chess GUI / test harness at bin/uci).
uci:
	go build -o bin/uci ./cmd/uci

# Run the test suite. `go vet` runs as part of `go test` (do not disable it:
# it catches printf mismatches, lost struct tags and bad mutex copies).
test:
	go test ./...

# Static analysis only.
vet:
	go vet ./...

run:
	go run .

# Engine-vs-engine match via fastchess (https://github.com/Disservin/fastchess).
#
# Point BOOK at a real opening book -- a few thousand balance-tested positions,
# e.g. UHO or Pohl. Do not use a handful of hand-picked openings: with too few
# positions the games repeat, and untested positions can be lopsided enough to
# bias the Elo. Books are maintained upstream and are not vendored here.
#
#   make bench ENGINE=bin/uci BASE=bin/uci-base BOOK=~/books/UHO_4060.epd
FASTCHESS ?= fastchess
ENGINE    ?= bin/uci
BASE      ?= bin/uci
TC        ?= 10+0.1
ROUNDS    ?= 500
bench: uci
ifndef BOOK
	$(error BOOK is not set. Pass a book, e.g. make bench BOOK=~/books/UHO_4060.epd)
endif
	$(FASTCHESS) \
	  -engine cmd=$(ENGINE) name=new \
	  -engine cmd=$(BASE) name=base \
	  -each tc=$(TC) -rounds $(ROUNDS) -repeat -concurrency 8 \
	  -openings file=$(BOOK) format=epd order=random \
	  -sprt elo0=0 elo1=10 alpha=0.05 beta=0.05 \
	  -pgnout file=bench.pgn

# Production stack (base compose only).
up:
	docker compose up -d --build

# Local dev stack with hot reload (base + dev override).
dev:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yml up --build
