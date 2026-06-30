package email

import "context"

// Message is a fully-rendered email ready to send.
type Message struct {
	From        string
	FromName    string
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	ReplyTo     string
	Tags        map[string]string // provider-level metadata tags
}

// SendResult is returned by a successful provider send.
type SendResult struct {
	ProviderMessageID string
}

// Provider is the interface every email adapter must satisfy.
type Provider interface {
	Send(ctx context.Context, msg *Message) (*SendResult, error)
	Name() string
}
