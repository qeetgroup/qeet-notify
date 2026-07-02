package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const twoFactorSendURL = "https://2factor.in/API/V1/%s/SMS/%s/%s/%s"

// TwoFactorProvider sends SMS via the 2Factor API (India fallback).
type TwoFactorProvider struct {
	apiKey string
	client *http.Client
}

// NewTwoFactor creates a TwoFactorProvider.
func NewTwoFactor(apiKey string) *TwoFactorProvider {
	return &TwoFactorProvider{apiKey: apiKey, client: &http.Client{}}
}

func (p *TwoFactorProvider) Name() string { return "2factor" }

func (p *TwoFactorProvider) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	endpoint := fmt.Sprintf(twoFactorSendURL,
		p.apiKey,
		url.PathEscape(msg.To),
		url.PathEscape(msg.Body),
		url.PathEscape(msg.SenderID),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build 2factor request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("2factor send: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("2factor %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		Status  string `json:"Status"`
		Details string `json:"Details"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse 2factor response: %w", err)
	}
	if out.Status != "Success" {
		return nil, fmt.Errorf("2factor error: %s", out.Details)
	}
	return &SendResult{ProviderMessageID: out.Details}, nil
}
