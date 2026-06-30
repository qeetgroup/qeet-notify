package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/qeetgroup/qeet-notify/internal/api/handler"
	"github.com/qeetgroup/qeet-notify/internal/platform/cache"
	"github.com/qeetgroup/qeet-notify/internal/platform/config"
	"github.com/qeetgroup/qeet-notify/internal/platform/db"
	"github.com/qeetgroup/qeet-notify/internal/platform/logger"
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

	rdb, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to redis")
	}
	defer rdb.Close()

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	// SSE stream and notification feed are tenant-scoped via path params.
	// Short-lived subscriber tokens (Step 10) will add JWT auth here.
	r.Get("/v1/tenants/{tenantID}/subscribers/{subscriberID}/stream",
		handler.NotificationStream(rdb))
	r.Get("/v1/tenants/{tenantID}/subscribers/{subscriberID}/notifications",
		handler.NotificationFeed(pool))
	r.Patch("/v1/tenants/{tenantID}/notifications/{notifID}/read",
		handler.MarkNotificationRead(pool))

	port := "8082"
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  0, // SSE streams need infinite read timeout
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("port", port).Msg("qeet-notify-sse starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("sse server error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("sse shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx) //nolint:errcheck
}
