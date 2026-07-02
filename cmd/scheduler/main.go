package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/qeetgroup/qeet-notify/domains/scheduler"
	"github.com/qeetgroup/qeet-notify/platform/config"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
	"github.com/qeetgroup/qeet-notify/platform/telemetry"
)

// scheduler re-enqueues delayed workflow runs once their resume_at has passed.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log := telemetry.NewLogger(cfg.Env)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}
	defer pool.Close()

	nc, err := messaging.New(cfg.NATSURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to NATS")
	}
	defer nc.Close()

	if err := nc.EnsureStreams(ctx); err != nil {
		log.Fatal().Err(err).Msg("ensure NATS streams")
	}

	if err := scheduler.New(pool, nc, log).Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("scheduler error")
	}
}
