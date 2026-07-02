package sms

import "context"

// Message is a ready-to-send SMS.
type Message struct {
	To         string            // E.164 format, e.g. "+919876543210"
	Body       string            // final rendered text
	SenderID   string            // TRAI-registered Sender Header (e.g. "QEETID")
	DLTTmplID  string            // TRAI DLT template ID
	IsUnicode  bool
	Tags       map[string]string
}

// SendResult is returned by a successful provider send.
type SendResult struct {
	ProviderMessageID string
}

// Provider is the interface every SMS adapter must satisfy.
type Provider interface {
	Send(ctx context.Context, msg *Message) (*SendResult, error)
	Name() string
}
