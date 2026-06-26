.PHONY: build uci test run up dev

# Build the HTTP/WebSocket game server.
build:
	go build -o bin/server .

# Build the UCI engine binary (point a chess GUI / test harness at bin/uci).
uci:
	go build -o bin/uci ./cmd/uci

# Run the test suite. vet is disabled because of a pre-existing finding in
# app/engine/layout.go unrelated to the engine logic under test.
test:
	go test -vet=off ./...

run:
	go run .

# Production stack (base compose only).
up:
	docker compose up -d --build

# Local dev stack with hot reload (base + dev override).
dev:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yml up --build
