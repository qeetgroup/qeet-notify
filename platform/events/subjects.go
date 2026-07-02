package events

import "fmt"

// NATS stream names — must match the stream definitions in platform/messaging/streams.go.
const (
	StreamEvents   = "NOTIFY_EVENTS"
	StreamEmail    = "NOTIFY_EMAIL"
	StreamSMS      = "NOTIFY_SMS"
	StreamWhatsApp = "NOTIFY_WHATSAPP"
	StreamPush     = "NOTIFY_PUSH"
	StreamInApp    = "NOTIFY_INAPP"
	StreamWebhook  = "NOTIFY_WEBHOOK"
	StreamDelivery = "NOTIFY_DELIVERY"
	StreamDLQ      = "NOTIFY_DLQ"
)

// EventSubject returns the intake subject for a tenant: "qeet-notify.<tenant>.events".
func EventSubject(tenantID string) string {
	return fmt.Sprintf("qeet-notify.%s.events", tenantID)
}

// ChannelSubject returns the per-channel work-queue subject for a tenant.
func ChannelSubject(tenantID, channel string) string {
	return fmt.Sprintf("qeet-notify.%s.channel.%s", tenantID, channel)
}

// DeliverySubject returns the delivery fan-out subject for a tenant.
func DeliverySubject(tenantID string) string {
	return fmt.Sprintf("qeet-notify.%s.delivery", tenantID)
}

// DLQSubject returns the dead-letter subject for a tenant.
func DLQSubject(tenantID string) string {
	return fmt.Sprintf("qeet-notify.%s.dlq", tenantID)
}
