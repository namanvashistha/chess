package main

import (
	"chess-engine/app/router"
	"chess-engine/config"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// defaultPort is used when PORT is unset. Previously the value was interpolated
// unchecked into ":" + port, so an unset PORT produced Run(":").
const defaultPort = "9000"

func init() {
	_ = godotenv.Load()
	config.InitLog()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Warnf("PORT is not set, defaulting to %s", defaultPort)
		port = defaultPort
	}

	// Named `deps`, not `init`: shadowing the identifier of this file's own
	// init() function is legal but actively confusing.
	deps := config.Init()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router.Init(deps),
	}

	// Shut down on SIGINT/SIGTERM so in-flight requests finish and the container
	// stops promptly instead of being killed.
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server failed to start: ", err)
		}
	}()
	log.Infof("Listening on :%s", port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Graceful shutdown failed: ", err)
	}
}
