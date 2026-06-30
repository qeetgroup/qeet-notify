package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

var streamDefs = []jetstream.StreamConfig{
	{Name: "NOTIFY_EVENTS",   Subjects: []string{"qeet-notify.*.events"},          Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_EMAIL",    Subjects: []string{"qeet-notify.*.channel.email"},    Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_SMS",      Subjects: []string{"qeet-notify.*.channel.sms"},      Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_WHATSAPP", Subjects: []string{"qeet-notify.*.channel.whatsapp"}, Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_PUSH",     Subjects: []string{"qeet-notify.*.channel.push"},     Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_INAPP",    Subjects: []string{"qeet-notify.*.channel.inapp"},    Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_WEBHOOK",  Subjects: []string{"qeet-notify.*.channel.webhook"},  Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * time.Hour},
	{Name: "NOTIFY_DELIVERY", Subjects: []string{"qeet-notify.*.delivery"},         Retention: jetstream.LimitsPolicy,    MaxAge: 30 * 24 * time.Hour},
}

// EnsureStreams creates or updates all streams idempotently.
func (c *Client) EnsureStreams(ctx context.Context) error {
	for _, def := range streamDefs {
		if _, err := c.JS.CreateOrUpdateStream(ctx, def); err != nil {
			return fmt.Errorf("ensure stream %s: %w", def.Name, err)
		}
	}
	return nil
}

// EventSubject returns the NATS subject for a tenant's event intake.
func EventSubject(tenantID string) string {
	return fmt.Sprintf("qeet-notify.%s.events", tenantID)
}

// ChannelSubject returns the NATS subject for a tenant's per-channel queue.
func ChannelSubject(tenantID, channel string) string {
	return fmt.Sprintf("qeet-notify.%s.channel.%s", tenantID, channel)
}

// DeliverySubject returns the NATS subject for delivery event fan-out.
func DeliverySubject(tenantID string) string {
	return fmt.Sprintf("qeet-notify.%s.delivery", tenantID)
}
