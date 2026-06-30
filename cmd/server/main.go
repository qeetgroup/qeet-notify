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
	"github.com/go-chi/cors"

	"github.com/qeetgroup/qeet-notify/internal/api/handler"
	apimw "github.com/qeetgroup/qeet-notify/internal/api/middleware"
	"github.com/qeetgroup/qeet-notify/internal/platform/cache"
	"github.com/qeetgroup/qeet-notify/internal/platform/config"
	"github.com/qeetgroup/qeet-notify/internal/platform/db"
	"github.com/qeetgroup/qeet-notify/internal/platform/logger"
	platformnats "github.com/qeetgroup/qeet-notify/internal/platform/nats"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Env)

	ctx := context.Background()

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

	nc, err := platformnats.New(cfg.NATSURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to NATS")
	}
	defer nc.Close()

	if err := nc.EnsureStreams(ctx); err != nil {
		log.Fatal().Err(err).Msg("ensure NATS streams")
	}

	tenantLookup := apimw.TenantLookup(func(ctx context.Context, keyHash string) (string, bool, error) {
		return db.LookupTenantByAPIKeyHash(ctx, pool, keyHash)
	})

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Qeet-Api-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", handler.Health)
	r.Get("/readyz", handler.Health)

	r.Route("/v1", func(r chi.Router) {
		r.Use(apimw.Auth(tenantLookup))
		r.Use(apimw.RateLimit(rdb, 1000, time.Minute))

		r.Post("/events", handler.NewTriggerEvent(nc.JS))

		// Subscriber management + DPDP erasure
		r.Get("/subscribers/{subscriberID}/preferences", handler.GetPreferences(pool))
		r.Delete("/subscribers/{subscriberID}", handler.DeleteSubscriber(pool))

		// Analytics
		r.Get("/analytics/delivery", handler.DeliveryAnalytics(pool))
	})

	// One-click unsubscribe — no API key (linked from emails).
	r.Get("/v1/unsubscribe", handler.Unsubscribe(pool))

	// Provider webhooks — no API key auth; providers call these directly.
	r.Post("/v1/webhooks/email/{provider}", handler.InboundEmailWebhook(pool))

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("qeet-notify api starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}
}
