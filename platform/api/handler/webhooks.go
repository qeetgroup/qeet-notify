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
)

// InboundEmailWebhook handles provider delivery/bounce/complaint callbacks.
// Supports: ses, resend.
func InboundEmailWebhook(pool *pgxpool.Pool) http.HandlerFunc {
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

		go processEmailEvent(context.Background(), pool, provider, providerMsgID, eventType)

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

func processEmailEvent(ctx context.Context, pool *pgxpool.Pool, provider, providerMsgID, eventType string) {
	// Look up the notification by provider_message_id.
	var notifID, tenantID, subscriberID, emailEnc string
	err := pool.QueryRow(ctx,
		`SELECT n.id, n.tenant_id, n.subscriber_id, COALESCE(s.email_encrypted,'')
		 FROM notifications n
		 LEFT JOIN subscribers s ON s.id = n.subscriber_id
		 WHERE n.provider_message_id = $1 AND n.provider = $2
		 LIMIT 1`,
		providerMsgID, provider,
	).Scan(&notifID, &tenantID, &subscriberID, &emailEnc)
	if err != nil {
		log.Error().Err(err).Str("provider_msg_id", providerMsgID).Msg("webhook: notification not found")
		return
	}

	// Record the delivery event.
	pool.Exec(ctx, //nolint:errcheck
		`INSERT INTO delivery_events (tenant_id, notification_id, event_type, provider, occurred_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		tenantID, notifID, eventType, provider, time.Now(),
	)

	// Hard bounces and spam complaints → add to suppressions.
	if (eventType == "bounced" || eventType == "complained") && emailEnc != "" {
		hash := sha256.Sum256([]byte(emailEnc))
		reason := "hard_bounce"
		if eventType == "complained" {
			reason = "spam_complaint"
		}
		pool.Exec(ctx, //nolint:errcheck
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
	pool.Exec(ctx, //nolint:errcheck
		`UPDATE notifications SET status = $1, updated_at = NOW() WHERE id = $2`,
		newStatus, notifID,
	)
}
