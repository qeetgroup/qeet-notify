package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

var (
	deliveryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "qeet_notify_delivery_total",
		Help: "Total notification delivery events by channel and event type.",
	}, []string{"channel", "event_type", "provider"})

	queueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "qeet_notify_queue_depth",
		Help: "Estimated NATS queue depth per channel.",
	}, []string{"channel"})
)

type deliveryEvent struct {
	NotificationID string `json:"notification_id"`
	TenantID       string `json:"tenant_id"`
	EventType      string `json:"event_type"`
	Provider       string `json:"provider"`
}

// Aggregator consumes NOTIFY_DELIVERY and writes to delivery_events + updates Prometheus.
type Aggregator struct {
	pool *pgxpool.Pool
	js   jetstream.JetStream
	log  zerolog.Logger
}

func New(pool *pgxpool.Pool, js jetstream.JetStream, log zerolog.Logger) *Aggregator {
	return &Aggregator{pool: pool, js: js, log: log}
}

func (a *Aggregator) Run(ctx context.Context) error {
	cons, err := a.js.CreateOrUpdateConsumer(ctx, "NOTIFY_DELIVERY", jetstream.ConsumerConfig{
		Name:          "analytics-aggregator",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxAckPending: 500,
	})
	if err != nil {
		return fmt.Errorf("create analytics consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe analytics: %w", err)
	}
	defer msgs.Stop()

	a.log.Info().Msg("analytics aggregator started")
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
			a.log.Error().Err(err).Msg("receive delivery event")
			continue
		}

		if err := a.handle(ctx, msg); err != nil {
			a.log.Error().Err(err).Msg("handle delivery event")
			msg.Nak() //nolint:errcheck
		} else {
			msg.Ack() //nolint:errcheck
		}
	}
}

func (a *Aggregator) handle(ctx context.Context, msg jetstream.Msg) error {
	var ev deliveryEvent
	if err := json.Unmarshal(msg.Data(), &ev); err != nil {
		return nil // malformed — ack to avoid infinite retry
	}

	// Fetch channel from notifications table for the Prometheus label.
	var channel string
	a.pool.QueryRow(ctx, //nolint:errcheck
		`SELECT channel FROM notifications WHERE id = $1`, ev.NotificationID,
	).Scan(&channel)
	if channel == "" {
		channel = "unknown"
	}

	// Write to delivery_events hypertable.
	_, err := a.pool.Exec(ctx,
		`INSERT INTO delivery_events (tenant_id, notification_id, event_type, provider, occurred_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		ev.TenantID, ev.NotificationID, ev.EventType, ev.Provider, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert delivery event: %w", err)
	}

	// Increment Prometheus counter.
	deliveryTotal.WithLabelValues(channel, ev.EventType, ev.Provider).Inc()
	return nil
}

// DeliveryStats is returned by the analytics query API.
type DeliveryStats struct {
	Channel    string  `json:"channel"`
	EventType  string  `json:"event_type"`
	Count      int64   `json:"count"`
	Date       string  `json:"date"`
}

// QueryDelivery returns 30-day delivery event counts bucketed by day, channel, and event_type.
func QueryDelivery(ctx context.Context, pool *pgxpool.Pool, tenantID string) ([]DeliveryStats, error) {
	rows, err := pool.Query(ctx,
		`SELECT
		    n.channel,
		    de.event_type,
		    COUNT(*) AS cnt,
		    date_trunc('day', de.occurred_at)::date::text AS day
		 FROM delivery_events de
		 JOIN notifications n ON n.id = de.notification_id
		 WHERE de.tenant_id = $1
		   AND de.occurred_at >= NOW() - INTERVAL '30 days'
		 GROUP BY n.channel, de.event_type, day
		 ORDER BY day DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("query delivery stats: %w", err)
	}
	defer rows.Close()

	var stats []DeliveryStats
	for rows.Next() {
		var s DeliveryStats
		if err := rows.Scan(&s.Channel, &s.EventType, &s.Count, &s.Date); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
