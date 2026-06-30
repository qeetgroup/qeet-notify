package email

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/templates/rendering"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// Worker consumes the NOTIFY_EMAIL JetStream queue and sends emails.
type Worker struct {
	pool     *pgxpool.Pool
	js       jetstream.JetStream
	primary  Provider
	fallback Provider // used when primary fails; may be nil
	log      zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, primary, fallback Provider, log zerolog.Logger) *Worker {
	return &Worker{pool: pool, js: js, primary: primary, fallback: fallback, log: log}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_EMAIL", jetstream.ConsumerConfig{
		Name:          "email-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxAckPending: 50,
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
			msg.Nak() //nolint:errcheck
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

	// Fetch subscriber email from DB.
	var emailEnc string
	err := w.pool.QueryRow(ctx,
		`SELECT COALESCE(email_encrypted, '') FROM subscribers WHERE id = $1 AND tenant_id = $2`,
		job.SubscriberID, job.TenantID,
	).Scan(&emailEnc)
	if err != nil || emailEnc == "" {
		return fmt.Errorf("fetch subscriber email: %w", err)
	}
	// TODO: decrypt emailEnc using pgp_sym_decrypt in a DB query with enc_key.
	// For now treat as plaintext in dev (migration adds encryption later).
	toEmail := emailEnc

	rendered, err := rendering.RenderEmail(ctx, w.pool, job.TenantID, job.TemplateID, job.Payload)
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

	result, providerName, sendErr := w.sendWithFallback(ctx, emailMsg)

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
	_, err = w.pool.Exec(ctx,
		`UPDATE notifications SET status = 'sent', provider = $1, provider_message_id = $2, updated_at = NOW()
		 WHERE id = $3`,
		providerName, result.ProviderMessageID, job.NotificationID,
	)
	return err
}

func (w *Worker) sendWithFallback(ctx context.Context, msg *Message) (*SendResult, string, error) {
	result, err := w.primary.Send(ctx, msg)
	if err == nil {
		return result, w.primary.Name(), nil
	}
	w.log.Warn().Err(err).Str("provider", w.primary.Name()).Msg("primary provider failed; trying fallback")

	if w.fallback == nil {
		return nil, w.primary.Name(), err
	}
	result, err = w.fallback.Send(ctx, msg)
	if err != nil {
		return nil, w.fallback.Name(), err
	}
	return result, w.fallback.Name(), nil
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
