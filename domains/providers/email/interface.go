// Package email defines the Provider contract for email delivery adapters.
// Each concrete implementation lives in a sub-package (ses/, resend/, sparkpost/).
//
// Migration note: domains/channels/email/ currently embeds the interface inline.
// Future work: update channels/email/worker.go to import from this package and
// remove the duplicate definitions from channels/.
package email

import "context"

// Message is a fully-rendered email ready to send.
type Message struct {
	From      string
	FromName  string
	To        string
	Subject   string
	HTMLBody  string
	TextBody  string
	ReplyTo   string
	Tags      map[string]string
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
