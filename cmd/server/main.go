// Command server runs the chess web app: the API, live games over WebSocket,
// and the built web client.
//
// Configuration comes from the environment:
//
//	PORT             listen port (default 9000)
//	DATABASE_URL     Postgres connection string (required)
//	ALLOWED_ORIGINS  extra origins allowed to use the API, comma-separated
//	TRUST_PROXY      "true" to read client addresses from X-Forwarded-For
//	LOG_LEVEL        debug, info (default), warn or error
//	WEB_DIR          built web app (default web/build)
//	STATIC_DIR       piece images (default app/static)
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"chess-engine/app/live"
	"chess-engine/app/server"
	"chess-engine/app/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel(os.Getenv("LOG_LEVEL"))}))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	db, err := store.OpenPostgres(startCtx, dbURL)
	cancel()
	if err != nil {
		return err
	}
	defer db.Close()

	rooms := live.NewManager(db, log, live.Options{})
	srv := &http.Server{
		Addr: ":" + env("PORT", "9000"),
		Handler: server.New(server.Config{
			Store:      db,
			Live:       rooms,
			Log:        log,
			WebDir:     env("WEB_DIR", "web/build"),
			Static:     env("STATIC_DIR", "app/static"),
			Origins:    splitList(os.Getenv("ALLOWED_ORIGINS")),
			TrustProxy: os.Getenv("TRUST_PROXY") == "true",
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", srv.Addr)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	rooms.Shutdown()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func logLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	return slog.LevelInfo
}
