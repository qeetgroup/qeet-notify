package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const metaAPIURL = "https://graph.facebook.com/v18.0/%s/messages"

// Category maps to Meta's message categories.
type Category string

const (
	CategoryUtility       Category = "UTILITY"
	CategoryAuthentication Category = "AUTHENTICATION"
	CategoryMarketing     Category = "MARKETING"
)

// Message is a ready-to-send WhatsApp message.
type Message struct {
	To           string   // E.164 without '+', e.g. "919876543210"
	TemplateName string   // Meta-approved template name
	LanguageCode string   // e.g. "en_US"
	Category     Category
	Components   []any // template component params (header/body/button)
}

// SendResult from a successful send.
type SendResult struct {
	ProviderMessageID string
}

// Provider is the interface every WhatsApp adapter must satisfy.
type Provider interface {
	Send(ctx context.Context, msg *Message) (*SendResult, error)
	Name() string
}

// MetaProvider sends WhatsApp messages via the Meta Cloud API.
type MetaProvider struct {
	token   string
	phoneID string
	client  *http.Client
}

func NewMeta(token, phoneID string) *MetaProvider {
	return &MetaProvider{token: token, phoneID: phoneID, client: &http.Client{}}
}

func (p *MetaProvider) Name() string { return "meta" }

func (p *MetaProvider) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	body := map[string]any{
		"messaging_product": "whatsapp",
		"to":                msg.To,
		"type":              "template",
		"template": map[string]any{
			"name":       msg.TemplateName,
			"language":   map[string]string{"code": msg.LanguageCode},
			"components": msg.Components,
		},
	}

	payload, _ := json.Marshal(body)
	url := fmt.Sprintf(metaAPIURL, p.phoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build meta wa request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta wa send: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("meta wa %d: %s", resp.StatusCode, raw)
	}

	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Messages) == 0 {
		return nil, fmt.Errorf("parse meta wa response: %w", err)
	}
	return &SendResult{ProviderMessageID: out.Messages[0].ID}, nil
}
