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

	"github.com/qeetgroup/qeet-notify/internal/india"
	platformnats "github.com/qeetgroup/qeet-notify/internal/platform/nats"
	notiftemplate "github.com/qeetgroup/qeet-notify/internal/template"
	"github.com/qeetgroup/qeet-notify/internal/workflow"
)

// errDeferred signals that the message was nak'd inside handle; Run must not ack/nak again.
var errDeferred = errors.New("message deferred")

type Worker struct {
	pool     *pgxpool.Pool
	js       jetstream.JetStream
	primary  Provider
	fallback Provider
	log      zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, primary, fallback Provider, log zerolog.Logger) *Worker {
	return &Worker{pool: pool, js: js, primary: primary, fallback: fallback, log: log}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_SMS", jetstream.ConsumerConfig{
		Name:          "sms-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxAckPending: 50,
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
			msg.Nak() //nolint:errcheck
		}
		// errDeferred: handle already called Nak; nothing to do here.
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var job workflow.ChannelJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return fmt.Errorf("unmarshal sms job: %w", err)
	}

	var phoneEnc string
	if err := w.pool.QueryRow(ctx,
		`SELECT COALESCE(phone_encrypted,'') FROM subscribers WHERE id = $1 AND tenant_id = $2`,
		job.SubscriberID, job.TenantID,
	).Scan(&phoneEnc); err != nil || phoneEnc == "" {
		return fmt.Errorf("fetch subscriber phone: %w", err)
	}
	phone := phoneEnc // TODO: pgp_sym_decrypt in production

	_, tmplBody, err := notiftemplate.Fetch(ctx, w.pool, job.TenantID, job.TemplateID)
	if err != nil {
		return err
	}
	rendered, err := notiftemplate.Render(tmplBody, job.Payload)
	if err != nil {
		return err
	}

	// DLT: load approved templates and match body.
	dltTemplates, err := india.LoadApprovedTemplates(ctx, w.pool, job.TenantID, "all")
	if err != nil {
		return err
	}
	matchedDLTID := india.MatchTemplate(dltTemplates, rendered)
	if matchedDLTID == "" {
		w.recordDelivery(ctx, job, "failed", "dlt_no_match", fmt.Errorf("no DLT template matched"))
		return nil // ack — operator needs to register/approve the template
	}

	// Promotional timing enforcement.
	var category string
	w.pool.QueryRow(ctx, `SELECT category FROM dlt_templates WHERE id = $1`, matchedDLTID).Scan(&category) //nolint:errcheck
	if category == "promotional" && !india.IsPromotionalWindowOpen() {
		resumeAt := india.ResumeAtNextWindow()
		w.log.Info().Time("resume_at", resumeAt).Msg("promotional SMS deferred")
		w.pool.Exec(ctx, //nolint:errcheck
			`UPDATE workflow_runs SET resume_at = $1, updated_at = NOW()
			 WHERE id = (SELECT workflow_run_id FROM notifications WHERE id = $2 LIMIT 1)`,
			resumeAt, job.NotificationID,
		)
		msg.Nak() //nolint:errcheck
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

func (w *Worker) recordDelivery(ctx context.Context, job workflow.ChannelJob, eventType, provider string, sendErr error) {
	payload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"event_type":      eventType,
		"provider":        provider,
	})
	_, _ = w.js.Publish(ctx, platformnats.DeliverySubject(job.TenantID), payload)
}

// DelayTicker re-dispatches workflow runs whose resume_at has passed.
func DelayTicker(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, log zerolog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			requeue(ctx, pool, js, log)
		}
	}
}

func requeue(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, log zerolog.Logger) {
	rows, err := pool.Query(ctx,
		`SELECT id, tenant_id FROM workflow_runs
		 WHERE status = 'running' AND resume_at IS NOT NULL AND resume_at <= NOW()
		 LIMIT 50`,
	)
	if err != nil {
		log.Error().Err(err).Msg("delay ticker query")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var runID, tenantID string
		if err := rows.Scan(&runID, &tenantID); err != nil {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"workflow_run_id": runID, "tenant_id": tenantID})
		_, _ = js.Publish(ctx, platformnats.EventSubject(tenantID), payload)
		pool.Exec(ctx, `UPDATE workflow_runs SET resume_at = NULL, updated_at = NOW() WHERE id = $1`, runID) //nolint:errcheck
	}
}
