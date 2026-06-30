package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/qeetgroup/qeet-notify/internal/platform/config"
	"github.com/qeetgroup/qeet-notify/internal/platform/db"
	"github.com/qeetgroup/qeet-notify/internal/platform/logger"
	platformnats "github.com/qeetgroup/qeet-notify/internal/platform/nats"
	"github.com/qeetgroup/qeet-notify/internal/workflow"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Env)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}
	defer pool.Close()

	nc, err := platformnats.New(cfg.NATSURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to NATS")
	}
	defer nc.Close()

	if err := nc.EnsureStreams(ctx); err != nil {
		log.Fatal().Err(err).Msg("ensure NATS streams")
	}

	engine := workflow.New(pool, nc, log)
	if err := engine.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("workflow engine error")
	}
}
