package sms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/compliance/dlt"
	"github.com/qeetgroup/qeet-notify/domains/templates/rendering"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// errDeferred signals that the message was nak'd inside handle; Run must not ack/nak again.
var errDeferred = errors.New("message deferred")

type Worker struct {
	pool     *pgxpool.Pool
	js       jetstream.JetStream
	primary  Provider
	fallback Provider
	encKey   string // pgcrypto key for decrypting subscriber PII
	log      zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, primary, fallback Provider, encKey string, log zerolog.Logger) *Worker {
	return &Worker{pool: pool, js: js, primary: primary, fallback: fallback, encKey: encKey, log: log}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_SMS", jetstream.ConsumerConfig{
		Name:          "sms-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxAckPending: 50,
		MaxDeliver:    messaging.DefaultMaxDeliver,
	})
	if err != nil {
		return fmt.Errorf("create sms consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe sms: %w", err)
	}
	defer msgs.Stop()

	w.log.Info().Msg("sms worker started")
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
			w.log.Error().Err(err).Msg("receive sms job")
			continue
		}

		err = w.handle(ctx, msg)
		if err == nil {
			msg.Ack() //nolint:errcheck
		} else if !errors.Is(err, errDeferred) {
			w.log.Error().Err(err).Msg("handle sms job")
			messaging.HandleFailure(ctx, w.js, msg, messaging.DefaultMaxDeliver, err, w.log)
		}
		// errDeferred: handle already called NakWithDelay; nothing to do here.
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var job engine.ChannelJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return fmt.Errorf("unmarshal sms job: %w", err)
	}

	var phone string
	if err := w.pool.QueryRow(ctx,
		`SELECT COALESCE(notify_decrypt(phone_encrypted, $3), '') FROM subscribers WHERE id = $1 AND tenant_id = $2`,
		job.SubscriberID, job.TenantID, w.encKey,
	).Scan(&phone); err != nil || phone == "" {
		return fmt.Errorf("fetch subscriber phone: %w", err)
	}

	_, tmplBody, err := rendering.Fetch(ctx, w.pool, job.TenantID, job.TemplateID)
	if err != nil {
		return err
	}
	rendered, err := rendering.Render(tmplBody, job.Payload)
	if err != nil {
		return err
	}

	// DLT: load approved templates and match body.
	dltTemplates, err := dlt.LoadApprovedTemplates(ctx, w.pool, job.TenantID, "all")
	if err != nil {
		return err
	}
	matchedDLTID := dlt.MatchTemplate(dltTemplates, rendered)
	if matchedDLTID == "" {
		w.recordDelivery(ctx, job, "failed", "dlt_no_match", fmt.Errorf("no DLT template matched"))
		return nil // ack — operator needs to register/approve the template
	}

	// Promotional timing enforcement.
	var category string
	w.pool.QueryRow(ctx, `SELECT category FROM dlt_templates WHERE id = $1`, matchedDLTID).Scan(&category) //nolint:errcheck
	if category == "promotional" && !dlt.IsPromotionalWindowOpen() {
		// Defer at the NATS layer: redeliver this exact job when the window opens.
		// Avoids mutating workflow_runs (which the scheduler now owns) and the
		// duplicate-notification risk of re-driving the whole workflow.
		delay := time.Until(dlt.ResumeAtNextWindow())
		w.log.Info().Dur("delay", delay).Msg("promotional SMS deferred to next window")
		msg.NakWithDelay(delay) //nolint:errcheck
		return errDeferred
	}

	var senderID string
	w.pool.QueryRow(ctx, `SELECT COALESCE(sender_id,'QEET') FROM dlt_templates WHERE id = $1`, matchedDLTID).Scan(&senderID) //nolint:errcheck

	smsMsg := &Message{
		To:        phone,
		Body:      rendered,
		SenderID:  senderID,
		DLTTmplID: matchedDLTID,
	}

	result, providerName, sendErr := w.sendWithFallback(ctx, smsMsg)
	eventType := "sent"
	if sendErr != nil {
		eventType = "failed"
	}
	w.recordDelivery(ctx, job, eventType, providerName, sendErr)
	if sendErr != nil {
		return sendErr
	}

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
	w.log.Warn().Err(err).Str("provider", w.primary.Name()).Msg("primary SMS provider failed")
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
	payload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"event_type":      eventType,
		"provider":        provider,
	})
	_, _ = w.js.Publish(ctx, messaging.DeliverySubject(job.TenantID), payload)
}
