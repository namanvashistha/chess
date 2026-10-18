.PHONY: build uci test vet run up dev bench ios-engine ios-upload

# This project has no cgo dependencies, and the Docker build already sets this.
# Keeping it off locally also avoids a Go 1.22 / recent-macOS link failure
# ("missing LC_UUID load command") in test binaries that pull in net.
export CGO_ENABLED = 0

# Build the HTTP/WebSocket game server.
build:
	go build -o bin/server ./cmd/server

# Build the UCI engine binary (point a chess GUI / test harness at bin/uci).
uci:
	go build -o bin/uci ./cmd/uci

# Run the test suite. `go vet` runs as part of `go test` (do not disable it:
# it catches printf mismatches, lost struct tags and bad mutex copies).
# Set TEST_DATABASE_URL to a throwaway database (its name must contain "test")
# to run the store tests against Postgres as well. CI also runs them with -race.
test:
	go test ./...

# Static analysis only.
vet:
	go vet ./...

run:
	go run ./cmd/server

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

# Build the engine into ios/Frameworks/Chesscore.xcframework for the iOS app.
# Needs Xcode and gomobile/gobind pinned to the same x/mobile as mobile/go.mod
# (newer x/mobile needs a newer Go). Build them without cgo, or macOS refuses
# to load them ("missing LC_UUID load command"):
#   CGO_ENABLED=0 go install golang.org/x/mobile/cmd/gomobile@v0.0.0-20240806205939-81131f6468ab
#   CGO_ENABLED=0 go install golang.org/x/mobile/cmd/gobind@v0.0.0-20240806205939-81131f6468ab
# The bind sets CGO_ENABLED back to 1: gomobile links the framework through cgo.
# gomobile and gobind live in Go's bin directory, which is often not on PATH.
ios-engine: export PATH := $(shell go env GOPATH)/bin:$(PATH)
ios-engine:
	cd mobile && CGO_ENABLED=0 go test ./... && CGO_ENABLED=1 gomobile bind -target=ios,iossimulator -iosversion=26.0 \
		-o ../ios/Frameworks/Chesscore.xcframework ./chesscore

# Archives Calm Chess and uploads it to App Store Connect, where it shows up
# in TestFlight. Bump CURRENT_PROJECT_VERSION in the project before each upload.
ios-upload: ios-engine
	cd ios && xcodebuild -project Chess.xcodeproj -scheme Chess -configuration Release \
		-destination 'generic/platform=iOS' -archivePath build/CalmChess.xcarchive -allowProvisioningUpdates archive
	cd ios && xcodebuild -exportArchive -archivePath build/CalmChess.xcarchive \
		-exportOptionsPlist ExportOptions.plist -exportPath build/export -allowProvisioningUpdates
