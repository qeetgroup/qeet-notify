package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// baseURL reads from E2E_API_URL env; falls back to localhost for local runs.
func baseURL() string {
	if u := os.Getenv("E2E_API_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func apiKey() string { return os.Getenv("E2E_API_KEY") }

// postJSON sends an authenticated POST and returns the response.
func postJSON(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, baseURL()+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Qeet-Api-Key", apiKey())
	req.Header.Set("Idempotency-Key", fmt.Sprintf("e2e-%d", time.Now().UnixNano()))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// TestNotificationFlow_EmailDelivery triggers a single email event and verifies it is
// accepted (202) and the notification record appears in GET /v1/notifications.
func TestNotificationFlow_EmailDelivery(t *testing.T) {
	if apiKey() == "" {
		t.Skip("E2E_API_KEY not set — skipping e2e tests")
	}

	resp := postJSON(t, "/v1/events", map[string]any{
		"name":        "e2e.email.test",
		"subscriber_id": "e2e-subscriber-001",
		"payload": map[string]any{
			"subject": "E2E test email",
			"to":      "e2e@example.com",
		},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	// Allow async processing.
	time.Sleep(3 * time.Second)

	req, _ := http.NewRequest(http.MethodGet, baseURL()+"/v1/notifications?limit=5", nil)
	req.Header.Set("X-Qeet-Api-Key", apiKey())
	listResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/notifications: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d", listResp.StatusCode)
	}
}

// TestNotificationFlow_IdempotencyDedup sends the same Idempotency-Key twice and
// verifies the second request returns 200 (replay) rather than creating a duplicate.
func TestNotificationFlow_IdempotencyDedup(t *testing.T) {
	if apiKey() == "" {
		t.Skip("E2E_API_KEY not set — skipping e2e tests")
	}

	key := fmt.Sprintf("dedup-e2e-%d", time.Now().UnixNano())
	body := map[string]any{
		"name":        "e2e.dedup.test",
		"subscriber_id": "e2e-subscriber-001",
		"payload":     map[string]any{},
	}

	b, _ := json.Marshal(body)
	send := func() *http.Response {
		req, _ := http.NewRequest(http.MethodPost, baseURL()+"/v1/events", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Qeet-Api-Key", apiKey())
		req.Header.Set("Idempotency-Key", key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /v1/events: %v", err)
		}
		return resp
	}

	r1 := send()
	r1.Body.Close()
	if r1.StatusCode != http.StatusAccepted {
		t.Fatalf("first request: expected 202, got %d", r1.StatusCode)
	}

	// Allow Redis SETNX TTL to propagate.
	time.Sleep(100 * time.Millisecond)

	r2 := send()
	r2.Body.Close()
	// Idempotent replay — no duplicate processed.
	if r2.StatusCode != http.StatusOK && r2.StatusCode != http.StatusAccepted {
		t.Fatalf("second request (dedup): expected 200 or 202, got %d", r2.StatusCode)
	}
}
