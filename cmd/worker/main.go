package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/qeetgroup/qeet-notify/domains/providers/email"
	"github.com/qeetgroup/qeet-notify/domains/providers/inapp"
	"github.com/qeetgroup/qeet-notify/domains/providers/sms"
	"github.com/qeetgroup/qeet-notify/domains/providers/webhook"
	"github.com/qeetgroup/qeet-notify/domains/providers/whatsapp"
	"github.com/qeetgroup/qeet-notify/platform/cache"
	"github.com/qeetgroup/qeet-notify/platform/config"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
	"github.com/qeetgroup/qeet-notify/platform/observability"
)

func main() {
	channel := flag.String("channel", "", "Channel to run: email|sms|whatsapp|push|webhook")
	flag.Parse()
	if *channel == "" {
		fmt.Fprintln(os.Stderr, "-channel required: email|sms|whatsapp|push|webhook")
		os.Exit(1)
	}

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

	switch *channel {
	case "email":
		var primary email.Provider
		if cfg.AWSSESAccessKey != "" {
			p, err := email.NewSES(cfg.AWSSESRegion, cfg.AWSSESAccessKey, cfg.AWSSESSecretKey)
			if err != nil {
				log.Fatal().Err(err).Msg("init SES")
			}
			primary = p
		} else {
			primary = email.NewResend(cfg.ResendAPIKey)
		}

		var fallback email.Provider
		if cfg.ResendAPIKey != "" && primary.Name() != "resend" {
			fallback = email.NewResend(cfg.ResendAPIKey)
		}

		w := email.NewWorker(pool, nc.JS, primary, fallback, log)
		if err := w.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("email worker")
		}

	case "sms":
		var primary sms.Provider
		if cfg.MSG91APIKey != "" {
			primary = sms.NewMSG91(cfg.MSG91APIKey)
		} else {
			primary = sms.NewTwoFactor(cfg.TwoFactorKey)
		}
		var fallback sms.Provider
		if cfg.TwoFactorKey != "" && primary.Name() != "2factor" {
			fallback = sms.NewTwoFactor(cfg.TwoFactorKey)
		}
		w := sms.NewWorker(pool, nc.JS, primary, fallback, log)
		if err := w.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("sms worker")
		}

	case "whatsapp":
		rdb, err := cache.New(cfg.RedisURL)
		if err != nil {
			log.Fatal().Err(err).Msg("connect to redis")
		}
		defer rdb.Close()
		provider := whatsapp.NewMeta(cfg.MetaWAToken, cfg.MetaWAPhoneID)
		w := whatsapp.NewWorker(pool, nc.JS, provider, rdb, log)
		if err := w.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("whatsapp worker")
		}

	case "inapp":
		rdb, err := cache.New(cfg.RedisURL)
		if err != nil {
			log.Fatal().Err(err).Msg("connect to redis")
		}
		defer rdb.Close()
		w := inapp.NewWorker(pool, nc.JS, rdb, log)
		if err := w.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("inapp worker")
		}

	case "webhook":
		w := webhook.NewWorker(pool, nc.JS, log)
		if err := w.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("webhook worker")
		}

	default:
		log.Fatal().Str("channel", *channel).Msg("channel worker not yet implemented")
	}
}
