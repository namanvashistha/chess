# syntax=docker/dockerfile:1

# The engine, packaged with lichess-bot so it can play on lichess.org.
#
# This is a separate image rather than an init container that copies the engine
# into a shared volume. That earlier arrangement worked, but the copier exits as
# soon as it finishes, and Komodo counts any non-running container as unhealthy
# -- so the whole chess stack sat permanently UNHEALTHY and raised a
# StackStateChange alert on every deploy. Baking the two files in costs one more
# image build and leaves nothing to exit.

# --- Stage 1: build the UCI engine ---
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -o /out/uci ./cmd/uci

# --- Stage 2: lichess-bot with the engine and config in place ---
# Upstream expects /lichess-bot/config to hold config.yml plus the engine
# binary. It declares no VOLUME there, so files baked in at build time survive.
FROM ghcr.io/lichess-bot-devs/lichess-bot:latest
COPY --from=build /out/uci /lichess-bot/config/uci
COPY deploy/lichess/config.yml /lichess-bot/config/config.yml
