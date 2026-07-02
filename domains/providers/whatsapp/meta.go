package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MetaProvider sends WhatsApp messages via the Meta Cloud API.
type MetaProvider struct {
	token   string
	phoneID string
	client  *http.Client
}

// NewMeta creates a MetaProvider.
func NewMeta(token, phoneID string) *MetaProvider {
	return &MetaProvider{token: token, phoneID: phoneID, client: &http.Client{}}
}

func (p *MetaProvider) Name() string { return "meta_whatsapp" }

func (p *MetaProvider) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", p.phoneID)
	body := map[string]any{
		"messaging_product": "whatsapp",
		"to":                msg.To,
	}
	if msg.TemplateName != "" {
		body["type"] = "template"
		body["template"] = map[string]any{
			"name":       msg.TemplateName,
			"language":   map[string]any{"code": msg.Language},
			"components": msg.Components,
		}
	} else {
		body["type"] = "text"
		body["text"] = map[string]any{"body": msg.Body}
	}
	payload, _ := json.Marshal(body)
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
