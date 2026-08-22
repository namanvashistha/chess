.PHONY: build uci test vet run up dev

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

# Production stack (base compose only).
up:
	docker compose up -d --build

# Local dev stack with hot reload (base + dev override).
dev:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yml up --build
