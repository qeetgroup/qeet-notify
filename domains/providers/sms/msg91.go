package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const msg91SendURL = "https://api.msg91.com/api/v5/flow/"

// MSG91Provider sends SMS via the MSG91 Flow API (India primary).
type MSG91Provider struct {
	apiKey string
	client *http.Client
}

// NewMSG91 creates a MSG91Provider.
func NewMSG91(apiKey string) *MSG91Provider {
	return &MSG91Provider{apiKey: apiKey, client: &http.Client{}}
}

func (p *MSG91Provider) Name() string { return "msg91" }

func (p *MSG91Provider) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	body := map[string]any{
		"template_id": msg.DLTTmplID,
		"sender":      msg.SenderID,
		"short_url":   "0",
		"recipients":  []map[string]any{{"mobiles": msg.To}},
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, msg91SendURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build msg91 request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authkey", p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("msg91 send: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("msg91 %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		Type      string `json:"type"`
		MessageID string `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse msg91 response: %w", err)
	}
	if out.Type == "error" {
		return nil, fmt.Errorf("msg91 error: %s", out.MessageID)
	}
	return &SendResult{ProviderMessageID: out.MessageID}, nil
}
