package events

// TriggerEvent is the canonical inbound event published by POST /v1/events.
// It is JSON-serialised onto the NATS NOTIFY_EVENTS work-queue stream.
type TriggerEvent struct {
	TenantID   string         `json:"tenant_id"`
	EventName  string         `json:"event"`
	SubscriberID string       `json:"subscriber_id"`
	Payload    map[string]any `json:"payload,omitempty"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
}

// DeliveryEvent is published to NOTIFY_DELIVERY after every send attempt.
type DeliveryEvent struct {
	TenantID       string  `json:"tenant_id"`
	NotificationID string  `json:"notification_id"`
	Channel        string  `json:"channel"`
	EventType      string  `json:"event_type"` // sent | failed | suppressed | ndnc_blocked
	Provider       string  `json:"provider,omitempty"`
	Error          *string `json:"error,omitempty"`
}

// DLQEvent is published to NOTIFY_DLQ when a job exhausts all retries.
type DLQEvent struct {
	TenantID       string `json:"tenant_id"`
	NotificationID string `json:"notification_id"`
	Channel        string `json:"channel"`
	OriginalSubject string `json:"original_subject"`
	LastError      string `json:"last_error"`
}
