package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/qeetgroup/qeet-notify/domains/analytics"
	"github.com/qeetgroup/qeet-notify/platform/config"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
	"github.com/qeetgroup/qeet-notify/platform/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log := observability.New(cfg.Env)
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

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		srv := &http.Server{Addr: ":9090", Handler: mux, ReadTimeout: 5 * time.Second}
		log.Info().Msg("metrics server starting on :9090")
		srv.ListenAndServe() //nolint:errcheck
	}()

	agg := analytics.New(pool, nc.JS, log)
	if err := agg.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("analytics aggregator error")
	}
}
