package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const resendSendURL = "https://api.resend.com/emails"

// ResendProvider sends email via the Resend API.
type ResendProvider struct {
	apiKey string
	client *http.Client
}

// NewResend creates a ResendProvider.
func NewResend(apiKey string) *ResendProvider {
	return &ResendProvider{apiKey: apiKey, client: &http.Client{}}
}

func (p *ResendProvider) Name() string { return "resend" }

func (p *ResendProvider) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	from := msg.From
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
	}
	body := map[string]any{
		"from": from, "to": []string{msg.To}, "subject": msg.Subject, "html": msg.HTMLBody,
	}
	if msg.TextBody != "" {
		body["text"] = msg.TextBody
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendSendURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resend request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("resend %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse resend response: %w", err)
	}
	return &SendResult{ProviderMessageID: out.ID}, nil
}
