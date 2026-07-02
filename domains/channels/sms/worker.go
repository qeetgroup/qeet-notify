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
	"github.com/qeetgroup/qeet-notify/domains/compliance/ndnc"
	"github.com/qeetgroup/qeet-notify/domains/routing"
	"github.com/qeetgroup/qeet-notify/domains/subscribers/preferences"
	"github.com/qeetgroup/qeet-notify/domains/templates/rendering"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/database"
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
	// Run the pipeline in a tenant-scoped tx so RLS applies (Module 36).
	var delay time.Duration
	err := database.RunInTenant(ctx, w.pool, job.TenantID, func(ctx context.Context, _ database.Querier) error {
		d, e := w.handleJob(ctx, job)
		delay = d
		return e
	})
	if err != nil {
		return err
	}
	if delay > 0 {
		// Defer at the NATS layer: redeliver this exact job when the window opens.
		// Avoids mutating workflow_runs (which the scheduler now owns) and the
		// duplicate-notification risk of re-driving the whole workflow.
		w.log.Info().Dur("delay", delay).Msg("promotional SMS deferred to next window")
		msg.NakWithDelay(delay) //nolint:errcheck
		return errDeferred
	}
	return nil
}

// handleJob runs the SMS send pipeline for a decoded job. It returns a non-zero
// delay when the message must be deferred (promotional window closed) — the
// caller performs the NATS NakWithDelay. Split from handle so the pipeline
// (including the NDNC/suppression gates) can be tested without a live message.
func (w *Worker) handleJob(ctx context.Context, job engine.ChannelJob) (time.Duration, error) {
	var phone string
	if err := database.FromContext(ctx, w.pool).QueryRow(ctx,
		`SELECT COALESCE(notify_decrypt(phone_encrypted, $3), '') FROM subscribers WHERE id = $1 AND tenant_id = $2`,
		job.SubscriberID, job.TenantID, w.encKey,
	).Scan(&phone); err != nil || phone == "" {
		return 0, fmt.Errorf("fetch subscriber phone: %w", err)
	}

	// Suppression check: never send to a suppressed number (Module 24).
	if suppressed, serr := preferences.IsSuppressed(ctx, database.FromContext(ctx, w.pool), job.TenantID, "sms", phone); serr != nil {
		return 0, fmt.Errorf("suppression check: %w", serr) // retry via NATS; do not send
	} else if suppressed {
		_, _ = database.FromContext(ctx, w.pool).Exec(ctx, `UPDATE notifications SET status = 'suppressed', updated_at = NOW() WHERE id = $1`, job.NotificationID)
		w.recordDelivery(ctx, job, "suppressed", "", nil)
		return 0, nil // ack; suppressed
	}

	_, tmplBody, err := rendering.Fetch(ctx, database.FromContext(ctx, w.pool), job.TenantID, job.TemplateID)
	if err != nil {
		return 0, err
	}
	rendered, err := rendering.Render(tmplBody, job.Payload)
	if err != nil {
		return 0, err
	}

	// DLT: load approved templates and match body.
	dltTemplates, err := dlt.LoadApprovedTemplates(ctx, database.FromContext(ctx, w.pool), job.TenantID, "all")
	if err != nil {
		return 0, err
	}
	matchedDLTID := dlt.MatchTemplate(dltTemplates, rendered)
	if matchedDLTID == "" {
		w.recordDelivery(ctx, job, "failed", "dlt_no_match", fmt.Errorf("no DLT template matched"))
		return 0, nil // ack — operator needs to register/approve the template
	}

	var category string
	database.FromContext(ctx, w.pool).QueryRow(ctx, `SELECT category FROM dlt_templates WHERE id = $1`, matchedDLTID).Scan(&category) //nolint:errcheck
	if category == "promotional" {
		// NDNC/DND scrub (Module 32): promotional traffic must never reach a
		// number on the national DND registry. Transactional is exempt.
		if dnd, derr := ndnc.IsRegistered(ctx, database.FromContext(ctx, w.pool), phone); derr != nil {
			return 0, fmt.Errorf("ndnc check: %w", derr) // retry via NATS; do not send
		} else if dnd {
			_, _ = database.FromContext(ctx, w.pool).Exec(ctx, `UPDATE notifications SET status = 'suppressed', updated_at = NOW() WHERE id = $1`, job.NotificationID)
			w.recordDelivery(ctx, job, "ndnc_blocked", "ndnc", nil)
			w.log.Info().Str("notification_id", job.NotificationID).Msg("promotional SMS blocked by NDNC")
			return 0, nil // ack; blocked
		}
		// Promotional timing enforcement (10:00–21:00 IST).
		if !dlt.IsPromotionalWindowOpen() {
			return time.Until(dlt.ResumeAtNextWindow()), nil
		}
	}

	var senderID string
	database.FromContext(ctx, w.pool).QueryRow(ctx, `SELECT COALESCE(sender_id,'QEET') FROM dlt_templates WHERE id = $1`, matchedDLTID).Scan(&senderID) //nolint:errcheck

	smsMsg := &Message{
		To:        phone,
		Body:      rendered,
		SenderID:  senderID,
		DLTTmplID: matchedDLTID,
	}

	result, providerName, sendErr := w.sendWithFallback(ctx, smsMsg, job.TenantID)
	eventType := "sent"
	if sendErr != nil {
		eventType = "failed"
	}
	w.recordDelivery(ctx, job, eventType, providerName, sendErr)
	if sendErr != nil {
		return 0, sendErr
	}

	_, err = database.FromContext(ctx, w.pool).Exec(ctx,
		`UPDATE notifications SET status = 'sent', provider = $1, provider_message_id = $2, updated_at = NOW()
		 WHERE id = $3`,
		providerName, result.ProviderMessageID, job.NotificationID,
	)
	return 0, err
}

func (w *Worker) sendWithFallback(ctx context.Context, msg *Message, tenantID string) (*SendResult, string, error) {
	providers := w.staticProviders()
	if records, err := routing.Load(ctx, database.FromContext(ctx, w.pool), tenantID, "sms", w.encKey); err != nil {
		w.log.Warn().Err(err).Msg("routing load failed; using static sms providers")
	} else if dbProviders, err := BuildProviders(records); err != nil {
		w.log.Warn().Err(err).Msg("routing build failed; using static sms providers")
	} else if len(dbProviders) > 0 {
		providers = dbProviders
	}

	var lastErr error
	for _, p := range providers {
		result, err := p.Send(ctx, msg)
		if err == nil {
			return result, p.Name(), nil
		}
		w.log.Warn().Err(err).Str("provider", p.Name()).Msg("sms provider failed; trying next")
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
	payload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"event_type":      eventType,
		"provider":        provider,
	})
	_, _ = w.js.Publish(ctx, messaging.DeliverySubject(job.TenantID), payload)
}
