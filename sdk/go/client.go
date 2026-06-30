// Package notify is the Go SDK for Qeet Notify.
//
// Usage:
//
//	client := notify.New("qn_live_xxx", notify.WithBaseURL("https://notify.api.qeet.in"))
//	err := client.Events.Trigger(ctx, "user.welcome", notify.TriggerParams{
//	    SubscriberID: "usr_123",
//	    Payload: map[string]any{"name": "Sai"},
//	})
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://notify.api.qeet.in"

// Client is the Qeet Notify API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	Events  *EventsService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL (useful for dev/staging).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// New creates a new Qeet Notify client.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{},
	}
	for _, o := range opts {
		o(c)
	}
	c.Events = &EventsService{client: c}
	return c
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Qeet-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("api %d: %s", resp.StatusCode, raw)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// TriggerParams are the arguments for Events.Trigger.
type TriggerParams struct {
	SubscriberID string         `json:"subscriber_id"`
	Payload      map[string]any `json:"payload,omitempty"`
}

// EventsService provides the Events API.
type EventsService struct {
	client *Client
}

// Trigger fires a named event for a subscriber, dispatching matching workflows.
func (s *EventsService) Trigger(ctx context.Context, event string, params TriggerParams) error {
	body := map[string]any{
		"event":         event,
		"subscriber_id": params.SubscriberID,
		"payload":       params.Payload,
	}
	return s.client.do(ctx, http.MethodPost, "/v1/events", body, nil)
}
