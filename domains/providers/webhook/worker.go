package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

const (
	maxRetries     = 5
	maxPayloadSize = 1 << 20 // 1 MB
)

// endpointConfig is loaded from provider_configs for channel=webhook.
type endpointConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// Worker processes NOTIFY_WEBHOOK jobs with HMAC signing and exponential retry.
type Worker struct {
	pool   *pgxpool.Pool
	js     jetstream.JetStream
	client *http.Client
	log    zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, log zerolog.Logger) *Worker {
	return &Worker{
		pool:   pool,
		js:     js,
		client: &http.Client{Timeout: 15 * time.Second},
		log:    log,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_WEBHOOK", jetstream.ConsumerConfig{
		Name:          "webhook-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       90 * time.Second,
		MaxAckPending: 50,
		MaxDeliver:    maxRetries,
	})
	if err != nil {
		return fmt.Errorf("create webhook consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe webhook: %w", err)
	}
	defer msgs.Stop()

	w.log.Info().Msg("webhook worker started")
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
			w.log.Error().Err(err).Msg("receive webhook job")
			continue
		}

		if err := w.handle(ctx, msg); err != nil {
			w.log.Error().Err(err).Msg("handle webhook job")
			// Exponential backoff via NATS AckWait — just nak.
			msg.Nak() //nolint:errcheck
		} else {
			msg.Ack() //nolint:errcheck
		}
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var job engine.ChannelJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return fmt.Errorf("unmarshal webhook job: %w", err)
	}

	// Load webhook endpoint config for this tenant.
	var configEnc string
	if err := w.pool.QueryRow(ctx,
		`SELECT config_encrypted FROM provider_configs
		 WHERE tenant_id = $1 AND channel = 'webhook' AND is_active
		 ORDER BY priority LIMIT 1`,
		job.TenantID,
	).Scan(&configEnc); err != nil {
		return fmt.Errorf("fetch webhook config: %w", err)
	}
	// TODO: decrypt configEnc; using plaintext JSON in dev.
	var cfg endpointConfig
	if err := json.Unmarshal([]byte(configEnc), &cfg); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	// Build signed payload.
	deliverPayload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"subscriber_id":   job.SubscriberID,
		"payload":         job.Payload,
		"timestamp":       time.Now().Unix(),
	})
	sig := hmacSign(cfg.Secret, deliverPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(deliverPayload))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Qeet-Signature-256", "sha256="+sig)
	req.Header.Set("X-Qeet-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook delivery: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, maxPayloadSize)) //nolint:errcheck

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook %d from %s", resp.StatusCode, cfg.URL)
	}

	// Record success delivery event.
	w.recordDelivery(ctx, job, "delivered", resp.StatusCode)

	_, err = w.pool.Exec(ctx,
		`UPDATE notifications SET status = 'delivered', updated_at = NOW() WHERE id = $1`,
		job.NotificationID,
	)
	return err
}

func (w *Worker) recordDelivery(ctx context.Context, job engine.ChannelJob, eventType string, statusCode int) {
	payload, _ := json.Marshal(map[string]any{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
		"event_type":      eventType,
		"provider":        "webhook",
		"provider_response": map[string]any{
			"status_code": statusCode,
		},
	})
	_, _ = w.js.Publish(ctx, messaging.DeliverySubject(job.TenantID), payload)
}

func hmacSign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
