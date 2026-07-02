package email

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/routing"
	"github.com/qeetgroup/qeet-notify/domains/subscribers/preferences"
	"github.com/qeetgroup/qeet-notify/domains/templates/rendering"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// Worker consumes the NOTIFY_EMAIL JetStream queue and sends emails.
type Worker struct {
	pool     *pgxpool.Pool
	js       jetstream.JetStream
	primary  Provider
	fallback Provider // used when primary fails; may be nil
	encKey   string   // pgcrypto key for decrypting subscriber PII
	log      zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, primary, fallback Provider, encKey string, log zerolog.Logger) *Worker {
	return &Worker{pool: pool, js: js, primary: primary, fallback: fallback, encKey: encKey, log: log}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_EMAIL", jetstream.ConsumerConfig{
		Name:          "email-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxAckPending: 50,
		MaxDeliver:    messaging.DefaultMaxDeliver,
	})
	if err != nil {
		return fmt.Errorf("create email consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe email: %w", err)
	}
	defer msgs.Stop()

	w.log.Info().Msg("email worker started")
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
			w.log.Error().Err(err).Msg("receive email job")
			continue
		}

		if err := w.handle(ctx, msg); err != nil {
			w.log.Error().Err(err).Msg("handle email job")
			messaging.HandleFailure(ctx, w.js, msg, messaging.DefaultMaxDeliver, err, w.log)
		} else {
			msg.Ack() //nolint:errcheck
		}
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var job engine.ChannelJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return fmt.Errorf("unmarshal job: %w", err)
	}
	// Run the pipeline in a tenant-scoped tx so RLS applies (Module 36). Inner
	// queries pick up the tx via database.FromContext.
	return database.RunInTenant(ctx, w.pool, job.TenantID, func(ctx context.Context, _ database.Querier) error {
		return w.handleJob(ctx, job)
	})
}

// handleJob runs the full send pipeline for a decoded job. Split out from handle
// so it can be exercised in tests without a live NATS message.
func (w *Worker) handleJob(ctx context.Context, job engine.ChannelJob) error {
	// Fetch + decrypt subscriber email from DB.
	var toEmail string
	err := database.FromContext(ctx, w.pool).QueryRow(ctx,
		`SELECT COALESCE(notify_decrypt(email_encrypted, $3), '') FROM subscribers WHERE id = $1 AND tenant_id = $2`,
		job.SubscriberID, job.TenantID, w.encKey,
	).Scan(&toEmail)
	if err != nil || toEmail == "" {
		return fmt.Errorf("fetch subscriber email: %w", err)
	}

	// Suppression check: never send to a suppressed address (Module 24).
	if suppressed, serr := preferences.IsSuppressed(ctx, database.FromContext(ctx, w.pool), job.TenantID, "email", toEmail); serr != nil {
		return fmt.Errorf("suppression check: %w", serr) // retry via NATS; do not send
	} else if suppressed {
		w.markSuppressed(ctx, job)
		return nil // ack; suppressed
	}

	rendered, err := rendering.RenderEmail(ctx, database.FromContext(ctx, w.pool), job.TenantID, job.TemplateID, job.Payload)
	if err != nil {
		return err
	}

	emailMsg := &Message{
		From:     "noreply@qeet.in",
		FromName: "Qeet Notify",
		To:       toEmail,
		Subject:  rendered.Subject,
		HTMLBody: rendered.Body,
		Tags:     map[string]string{"notification_id": job.NotificationID, "tenant_id": job.TenantID},
	}

	result, providerName, sendErr := w.sendWithFallback(ctx, emailMsg, job.TenantID)

	// Record delivery event regardless of success/failure.
	eventType := "sent"
	if sendErr != nil {
		eventType = "failed"
	}
	w.recordDelivery(ctx, job, eventType, providerName, sendErr)

	if sendErr != nil {
		return sendErr
	}

	// Update notification status + provider message ID.
	_, err = database.FromContext(ctx, w.pool).Exec(ctx,
		`UPDATE notifications SET status = 'sent', provider = $1, provider_message_id = $2, updated_at = NOW()
		 WHERE id = $3`,
		providerName, result.ProviderMessageID, job.NotificationID,
	)
	return err
}

func (w *Worker) sendWithFallback(ctx context.Context, msg *Message, tenantID string) (*SendResult, string, error) {
	// Prefer tenant-specific providers from DB; fall back to static startup config.
	providers := w.staticProviders()
	if records, err := routing.Load(ctx, database.FromContext(ctx, w.pool), tenantID, "email", w.encKey); err != nil {
		w.log.Warn().Err(err).Msg("routing load failed; using static email providers")
	} else if dbProviders, err := BuildProviders(records); err != nil {
		w.log.Warn().Err(err).Msg("routing build failed; using static email providers")
	} else if len(dbProviders) > 0 {
		providers = dbProviders
	}

	var lastErr error
	for _, p := range providers {
		result, err := p.Send(ctx, msg)
		if err == nil {
			return result, p.Name(), nil
		}
		w.log.Warn().Err(err).Str("provider", p.Name()).Msg("email provider failed; trying next")
		lastErr = err
	}
	name := ""
	if len(providers) > 0 {
		name = providers[len(providers)-1].Name()
	}
	return nil, name, lastErr
}

func (w *Worker) staticProviders() []Provider {
	if w.primary == nil {
		return nil
	}
	providers := []Provider{w.primary}
	if w.fallback != nil {
		providers = append(providers, w.fallback)
	}
	return providers
}

func (w *Worker) recordDelivery(ctx context.Context, job engine.ChannelJob, eventType, provider string, sendErr error) {
	var errStr *string
	if sendErr != nil {
		s := sendErr.Error()
		errStr = &s
	}
	subject := messaging.DeliverySubject(job.TenantID)
	payload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"event_type":      eventType,
		"provider":        provider,
		"error":           errStr,
	})
	_, _ = w.js.Publish(ctx, subject, payload)
}

// markSuppressed flags a notification that was blocked by the suppression list
// and records a "suppressed" delivery event (no provider send occurs).
func (w *Worker) markSuppressed(ctx context.Context, job engine.ChannelJob) {
	_, _ = database.FromContext(ctx, w.pool).Exec(ctx,
		`UPDATE notifications SET status = 'suppressed', updated_at = NOW() WHERE id = $1`,
		job.NotificationID,
	)
	w.recordDelivery(ctx, job, "suppressed", "", nil)
	w.log.Info().Str("notification_id", job.NotificationID).Msg("email suppressed")
}
