// Package push defines the Provider contract for push notification adapters.
// Concrete implementations live in sub-packages (fcm/, apns/).
// This channel is not yet live — stubs are present for future implementation.
package push

import "context"

// Message is a ready-to-send push notification.
type Message struct {
	DeviceToken string
	Title       string
	Body        string
	Data        map[string]string // key-value pairs for deep-linking / data payload
	BadgeCount  int
	Sound       string
}

// SendResult is returned by a successful provider send.
type SendResult struct {
	ProviderMessageID string
}

// Provider is the interface every push adapter must satisfy.
type Provider interface {
	Send(ctx context.Context, msg *Message) (*SendResult, error)
	Name() string
}
