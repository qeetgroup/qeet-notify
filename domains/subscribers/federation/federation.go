package federation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// qeet-id publishes user lifecycle events on these subjects.
// qeet-notify subscribes to upsert/delete subscribers automatically.
const (
	subjectUserCreated = "qeet-id.*.user.created"
	subjectUserUpdated = "qeet-id.*.user.updated"
	subjectUserDeleted = "qeet-id.*.user.deleted"
)

type qeetIDUserEvent struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
}

// Federate subscribes to qeet-id NATS events and mirrors users as subscribers.
// encKey is the pgcrypto key used to encrypt subscriber email/phone at rest.
func Federate(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, encKey string, log zerolog.Logger) error {
	// Use a core NATS subscription (not JetStream) since qeet-id publishes on core.
	// If qeet-id is co-located, this is a no-op — federation is best-effort.
	log.Info().Msg("subscriber federation: listening for qeet-id user events")

	// Subscribe via a durable consumer if the qeet-id stream exists.
	cons, err := js.CreateOrUpdateConsumer(ctx, "QEETID_USERS", jetstream.ConsumerConfig{
		Name:            "qeet-notify-federation",
		FilterSubjects:  []string{subjectUserCreated, subjectUserUpdated, subjectUserDeleted},
		AckPolicy:       jetstream.AckExplicitPolicy,
		MaxAckPending:   100,
	})
	if err != nil {
		// qeet-id stream may not exist yet; log and return without error.
		log.Warn().Err(err).Msg("qeet-id stream not available; subscriber federation disabled")
		return nil
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe to qeet-id events: %w", err)
	}
	defer msgs.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := msgs.Next()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		var ev qeetIDUserEvent
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			msg.Ack() //nolint:errcheck
			continue
		}

		subject := msg.Subject()
		// Each upsert/delete runs in a tenant-scoped tx so RLS applies (Module 36).
		switch {
		case matchesSuffix(subject, "user.created"), matchesSuffix(subject, "user.updated"):
			_ = database.RunInTenant(ctx, pool, ev.TenantID, func(ctx context.Context, q database.Querier) error {
				upsertSubscriber(ctx, q, ev, encKey, log)
				return nil
			})
		case matchesSuffix(subject, "user.deleted"):
			_ = database.RunInTenant(ctx, pool, ev.TenantID, func(ctx context.Context, q database.Querier) error {
				softDeleteSubscriber(ctx, q, ev.TenantID, ev.UserID, log)
				return nil
			})
		}
		msg.Ack() //nolint:errcheck
	}
}

func upsertSubscriber(ctx context.Context, q database.Querier, ev qeetIDUserEvent, encKey string, log zerolog.Logger) {
	locale := ev.Locale
	if locale == "" {
		locale = "en"
	}
	tz := ev.Timezone
	if tz == "" {
		tz = "Asia/Kolkata"
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO subscribers (tenant_id, external_id, email_encrypted, phone_encrypted, locale, timezone)
		 VALUES ($1, $2,
		         CASE WHEN $3::text IS NULL THEN NULL ELSE pgp_sym_encrypt($3::text, $7)::text END,
		         CASE WHEN $4::text IS NULL THEN NULL ELSE pgp_sym_encrypt($4::text, $7)::text END,
		         $5, $6)
		 ON CONFLICT (tenant_id, external_id) DO UPDATE
		   SET email_encrypted = EXCLUDED.email_encrypted,
		       phone_encrypted = EXCLUDED.phone_encrypted,
		       locale          = EXCLUDED.locale,
		       timezone        = EXCLUDED.timezone,
		       updated_at      = NOW()`,
		ev.TenantID, ev.UserID, nilStr(ev.Email), nilStr(ev.Phone), locale, tz, encKey,
	); err != nil {
		log.Error().Err(err).Str("user_id", ev.UserID).Msg("federation upsert failed")
	}
}

func softDeleteSubscriber(ctx context.Context, q database.Querier, tenantID, userID string, log zerolog.Logger) {
	if _, err := q.Exec(ctx,
		`UPDATE subscribers SET is_deleted = TRUE, deleted_at = NOW(), updated_at = NOW()
		 WHERE tenant_id = $1 AND external_id = $2`,
		tenantID, userID,
	); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("federation delete failed")
	}
}

func matchesSuffix(subject, suffix string) bool {
	l := len(subject)
	sl := len(suffix)
	return l >= sl && subject[l-sl:] == suffix
}

func nilStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
