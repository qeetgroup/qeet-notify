package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// InboundEmailWebhook handles provider delivery/bounce/complaint callbacks.
// Supports: ses, resend.
func InboundEmailWebhook(pool *pgxpool.Pool, encKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := chi.URLParam(r, "provider")
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB max
		if err != nil {
			http.Error(w, `{"error":"read body"}`, http.StatusBadRequest)
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		eventType, providerMsgID := extractEmailEvent(provider, payload)
		if eventType == "" || providerMsgID == "" {
			w.WriteHeader(http.StatusOK) // not a type we handle — ack silently
			return
		}

		go processEmailEvent(context.Background(), pool, encKey, provider, providerMsgID, eventType)

		w.WriteHeader(http.StatusOK)
	}
}

func extractEmailEvent(provider string, payload map[string]any) (eventType, msgID string) {
	switch provider {
	case "ses":
		// SNS-wrapped SES notification
		if msg, ok := payload["Message"].(string); ok {
			var inner map[string]any
			if err := json.Unmarshal([]byte(msg), &inner); err == nil {
				payload = inner
			}
		}
		notifType, _ := payload["notificationType"].(string)
		switch notifType {
		case "Bounce":
			if b, ok := payload["bounce"].(map[string]any); ok {
				if recipients, ok := b["bouncedRecipients"].([]any); ok && len(recipients) > 0 {
					if r0, ok := recipients[0].(map[string]any); ok {
						msgID, _ = r0["messageId"].(string)
					}
				}
			}
			return "bounced", msgID
		case "Complaint":
			return "complained", msgID
		case "Delivery":
			if mail, ok := payload["mail"].(map[string]any); ok {
				msgID, _ = mail["messageId"].(string)
			}
			return "delivered", msgID
		}
	case "resend":
		msgID, _ = payload["email_id"].(string)
		switch payload["type"] {
		case "email.bounced":
			return "bounced", msgID
		case "email.complained":
			return "complained", msgID
		case "email.delivered":
			return "delivered", msgID
		case "email.opened":
			return "opened", msgID
		case "email.clicked":
			return "clicked", msgID
		}
	}
	return "", ""
}

func processEmailEvent(ctx context.Context, pool *pgxpool.Pool, encKey, provider, providerMsgID, eventType string) {
	// Cross-tenant lookup of the notification by provider_message_id (the webhook
	// has no tenant context). notifications is not RLS-forced, so this works on
	// the shared pool; it yields the tenant for the scoped work below.
	var notifID, tenantID, subscriberID string
	err := pool.QueryRow(ctx,
		`SELECT id, tenant_id, subscriber_id FROM notifications
		 WHERE provider_message_id = $1 AND provider = $2
		 LIMIT 1`,
		providerMsgID, provider,
	).Scan(&notifID, &tenantID, &subscriberID)
	if err != nil {
		log.Error().Err(err).Str("provider_msg_id", providerMsgID).Msg("webhook: notification not found")
		return
	}

	// Everything else is tenant-scoped (subscribers + suppressions are RLS-forced).
	_ = database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, q database.Querier) error {
		// Decrypt the email so the suppression hash is computed over plaintext
		// (matching preferences.hashValue).
		var emailPlain string
		q.QueryRow(ctx, //nolint:errcheck
			`SELECT COALESCE(notify_decrypt(email_encrypted, $2), '') FROM subscribers WHERE id = $1`,
			subscriberID, encKey,
		).Scan(&emailPlain)

		// Record the delivery event.
		q.Exec(ctx, //nolint:errcheck
			`INSERT INTO delivery_events (tenant_id, notification_id, event_type, provider, occurred_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			tenantID, notifID, eventType, provider, time.Now(),
		)

		// Hard bounces and spam complaints → add to suppressions.
		if (eventType == "bounced" || eventType == "complained") && emailPlain != "" {
			hash := sha256.Sum256([]byte(emailPlain))
			reason := "hard_bounce"
			if eventType == "complained" {
				reason = "spam_complaint"
			}
			q.Exec(ctx, //nolint:errcheck
				`INSERT INTO suppressions (tenant_id, channel, value_hash, reason)
				 VALUES ($1, 'email', $2, $3)
				 ON CONFLICT DO NOTHING`,
				tenantID, hex.EncodeToString(hash[:]), reason,
			)
		}

		// Update notification status.
		newStatus := eventType
		if eventType == "bounced" || eventType == "complained" {
			newStatus = "failed"
		}
		q.Exec(ctx, //nolint:errcheck
			`UPDATE notifications SET status = $1, updated_at = NOW() WHERE id = $2`,
			newStatus, notifID,
		)
		return nil
	})
}
