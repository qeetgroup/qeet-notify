package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/qeetgroup/qeet-notify/platform/api/handler"
	apimw "github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/cache"
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

	rdb, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to redis")
	}
	defer rdb.Close()

	nc, err := messaging.New(cfg.NATSURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to NATS")
	}
	defer nc.Close()

	if err := nc.EnsureStreams(ctx); err != nil {
		log.Fatal().Err(err).Msg("ensure NATS streams")
	}

	tenantLookup := apimw.TenantLookup(func(ctx context.Context, keyHash string) (string, bool, error) {
		return database.LookupTenantByAPIKeyHash(ctx, pool, keyHash)
	})

	// API router (8080) — authenticated, standard timeouts.
	api := chi.NewRouter()
	api.Use(chimw.RequestID)
	api.Use(chimw.RealIP)
	api.Use(chimw.Recoverer)
	api.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Qeet-Api-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	api.Get("/healthz", handler.Health)
	api.Get("/readyz", handler.Health)

	api.Route("/v1", func(r chi.Router) {
		r.Use(apimw.Auth(tenantLookup))
		r.Use(apimw.RateLimit(rdb, 1000, time.Minute))

		r.Post("/events", handler.NewTriggerEvent(nc.JS))

		r.Get("/subscribers/{subscriberID}/preferences", handler.GetPreferences(pool))
		r.Delete("/subscribers/{subscriberID}", handler.DeleteSubscriber(pool))

		r.Get("/analytics/delivery", handler.DeliveryAnalytics(pool))
	})

	api.Get("/v1/unsubscribe", handler.Unsubscribe(pool))
	api.Post("/v1/webhooks/email/{provider}", handler.InboundEmailWebhook(pool))

	// SSE router (8082) — unauthenticated, infinite timeouts for streaming.
	sse := chi.NewRouter()
	sse.Use(chimw.RequestID)
	sse.Use(chimw.RealIP)
	sse.Use(chimw.Recoverer)

	sse.Get("/v1/tenants/{tenantID}/subscribers/{subscriberID}/stream",
		handler.NotificationStream(rdb))
	sse.Get("/v1/tenants/{tenantID}/subscribers/{subscriberID}/notifications",
		handler.NotificationFeed(pool))
	sse.Patch("/v1/tenants/{tenantID}/notifications/{notifID}/read",
		handler.MarkNotificationRead(pool))

	apiSrv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      api,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	sseSrv := &http.Server{
		Addr:         ":8082",
		Handler:      sse,
		ReadTimeout:  0, // SSE streams require infinite timeouts
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Info().Str("port", cfg.HTTPPort).Msg("qeet-notify api starting")
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("api server error")
		}
	}()
	go func() {
		defer wg.Done()
		log.Info().Str("port", "8082").Msg("qeet-notify sse starting")
		if err := sseSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("sse server error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	apiSrv.Shutdown(shutCtx) //nolint:errcheck
	sseSrv.Shutdown(shutCtx) //nolint:errcheck
	wg.Wait()
}
