package inapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// Worker consumes NOTIFY_INAPP jobs, persists them to the DB,
// and publishes to Redis pub/sub for SSE fan-out.
type Worker struct {
	pool *pgxpool.Pool
	js   jetstream.JetStream
	rdb  *redis.Client
	log  zerolog.Logger
}

func NewWorker(pool *pgxpool.Pool, js jetstream.JetStream, rdb *redis.Client, log zerolog.Logger) *Worker {
	return &Worker{pool: pool, js: js, rdb: rdb, log: log}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, "NOTIFY_INAPP", jetstream.ConsumerConfig{
		Name:          "inapp-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
		MaxDeliver:    messaging.DefaultMaxDeliver,
	})
	if err != nil {
		return fmt.Errorf("create inapp consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe inapp: %w", err)
	}
	defer msgs.Stop()

	w.log.Info().Msg("inapp worker started")
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
			w.log.Error().Err(err).Msg("receive inapp job")
			continue
		}

		if err := w.handle(ctx, msg); err != nil {
			w.log.Error().Err(err).Msg("handle inapp job")
			messaging.HandleFailure(ctx, w.js, msg, messaging.DefaultMaxDeliver, err, w.log)
		} else {
			msg.Ack() //nolint:errcheck
		}
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var job engine.ChannelJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return fmt.Errorf("unmarshal inapp job: %w", err)
	}

	// Mark notification as sent in DB.
	if _, err := w.pool.Exec(ctx,
		`UPDATE notifications SET status = 'sent', provider = 'inapp', updated_at = NOW()
		 WHERE id = $1`,
		job.NotificationID,
	); err != nil {
		return fmt.Errorf("update notification: %w", err)
	}

	// Fan-out to Redis pub/sub so SSE service pushes to active connections.
	channel := fmt.Sprintf("notify:inapp:%s:%s", job.TenantID, job.SubscriberID)
	pushPayload, _ := json.Marshal(map[string]string{
		"notification_id": job.NotificationID,
		"tenant_id":       job.TenantID,
	})
	return w.rdb.Publish(ctx, channel, pushPayload).Err()
}
