package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/routing"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// Worker processes NOTIFY_WHATSAPP jobs via Meta Cloud API.
type Worker struct {
	pool     *pgxpool.Pool
	js       jetstream.JetStream
	provider Provider // static fallback; may be nil
	encKey   string
	rdb      *redis.Client
	log      zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, provider Provider, encKey string, rdb *redis.Client, log zerolog.Logger) *Worker {
	return &Worker{pool: pool, js: js, provider: provider, encKey: encKey, rdb: rdb, log: log}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_WHATSAPP", jetstream.ConsumerConfig{
		Name:          "whatsapp-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxAckPending: 50,
		MaxDeliver:    messaging.DefaultMaxDeliver,
	})
	if err != nil {
		return fmt.Errorf("create whatsapp consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe whatsapp: %w", err)
	}
	defer msgs.Stop()

	w.log.Info().Msg("whatsapp worker started")
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
			w.log.Error().Err(err).Msg("receive whatsapp job")
			continue
		}

		if err := w.handle(ctx, msg); err != nil {
			w.log.Error().Err(err).Msg("handle whatsapp job")
			messaging.HandleFailure(ctx, w.js, msg, messaging.DefaultMaxDeliver, err, w.log)
		} else {
			msg.Ack() //nolint:errcheck
		}
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var job engine.ChannelJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return fmt.Errorf("unmarshal wa job: %w", err)
	}

	// Fetch subscriber whatsapp_id.
	var waID string
	if err := w.pool.QueryRow(ctx,
		`SELECT COALESCE(whatsapp_id,'') FROM subscribers WHERE id = $1 AND tenant_id = $2`,
		job.SubscriberID, job.TenantID,
	).Scan(&waID); err != nil || waID == "" {
		return fmt.Errorf("fetch subscriber wa_id: %w", err)
	}

	// Fetch DLT template metadata (category, carrier='meta').
	var tmplName, langCode, categoryStr string
	if err := w.pool.QueryRow(ctx,
		`SELECT template_id_ext, COALESCE(metadata->>'language_code','en_US'), category
		 FROM dlt_templates
		 WHERE id = $1 AND tenant_id = $2`,
		job.TemplateID, job.TenantID,
	).Scan(&tmplName, &langCode, &categoryStr); err != nil {
		return fmt.Errorf("fetch wa template: %w", err)
	}

	// Marketing messages require an open 24h service window.
	if Category(categoryStr) == CategoryMarketing {
		sessionKey := fmt.Sprintf("wa:session:%s:%s", job.SubscriberID, job.TenantID)
		exists, _ := w.rdb.Exists(ctx, sessionKey).Result()
		if exists == 0 {
			// No recent user message — skip marketing send.
			w.recordDelivery(ctx, job, "skipped", "no_wa_session")
			return nil
		}
	}

	waMsg := &Message{
		To:           waID,
		TemplateName: tmplName,
		LanguageCode: langCode,
		Category:     Category(categoryStr),
		Components:   buildComponents(job.Payload),
	}

	result, providerName, sendErr := w.sendWithFallback(ctx, waMsg, job.TenantID)
	if sendErr != nil {
		w.recordDelivery(ctx, job, "failed", providerName)
		return sendErr
	}

	w.recordDelivery(ctx, job, "sent", providerName)
	_, execErr := w.pool.Exec(ctx,
		`UPDATE notifications SET status = 'sent', provider = $1, provider_message_id = $2, updated_at = NOW()
		 WHERE id = $3`,
		providerName, result.ProviderMessageID, job.NotificationID,
	)
	return execErr
}

func (w *Worker) sendWithFallback(ctx context.Context, msg *Message, tenantID string) (*SendResult, string, error) {
	var providers []Provider
	if records, err := routing.Load(ctx, w.pool, tenantID, "whatsapp", w.encKey); err != nil {
		w.log.Warn().Err(err).Msg("routing load failed; using static whatsapp provider")
	} else if dbProviders, err := BuildProviders(records); err != nil {
		w.log.Warn().Err(err).Msg("routing build failed; using static whatsapp provider")
	} else if len(dbProviders) > 0 {
		providers = dbProviders
	}
	if len(providers) == 0 && w.provider != nil {
		providers = []Provider{w.provider}
	}

	var lastErr error
	for _, p := range providers {
		result, err := p.Send(ctx, msg)
		if err == nil {
			return result, p.Name(), nil
		}
		w.log.Warn().Err(err).Str("provider", p.Name()).Msg("whatsapp provider failed; trying next")
		lastErr = err
	}
	name := ""
	if len(providers) > 0 {
		name = providers[len(providers)-1].Name()
	}
	return nil, name, lastErr
}

// InboundWebhook updates the 24h service window key when a user sends a message.
func (w *Worker) InboundWebhook(ctx context.Context, subscriberID, tenantID string) {
	key := fmt.Sprintf("wa:session:%s:%s", subscriberID, tenantID)
	w.rdb.Set(ctx, key, "1", 24*time.Hour) //nolint:errcheck
}

func (w *Worker) recordDelivery(ctx context.Context, job engine.ChannelJob, eventType, provider string) {
	payload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"event_type":      eventType,
		"provider":        provider,
	})
	_, _ = w.js.Publish(ctx, messaging.DeliverySubject(job.TenantID), payload)
}

func buildComponents(payload map[string]any) []any {
	// Extract body parameters from payload["params"] if present.
	params, ok := payload["params"].([]any)
	if !ok {
		return nil
	}
	textParams := make([]map[string]string, 0, len(params))
	for _, p := range params {
		if s, ok := p.(string); ok {
			textParams = append(textParams, map[string]string{"type": "text", "text": s})
		}
	}
	if len(textParams) == 0 {
		return nil
	}
	return []any{
		map[string]any{"type": "body", "parameters": textParams},
	}
}
