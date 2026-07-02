// Package whatsapp defines the Provider contract for WhatsApp delivery adapters.
// Concrete implementations live in sub-packages (meta/).
package whatsapp

import "context"

// Message is a ready-to-send WhatsApp message.
type Message struct {
	To           string // E.164 format
	TemplateName string // WhatsApp template name (must be approved by Meta)
	Language     string // BCP-47 language code e.g. "en_US"
	Components   []map[string]any
	Body         string // plain-text fallback / non-template body
}

// SendResult is returned by a successful provider send.
type SendResult struct {
	ProviderMessageID string
}

// Provider is the interface every WhatsApp adapter must satisfy.
type Provider interface {
	Send(ctx context.Context, msg *Message) (*SendResult, error)
	Name() string
}
