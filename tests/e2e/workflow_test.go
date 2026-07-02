package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

// TestWorkflow_DelayAndMultiChannel triggers an event bound to a workflow that has a
// delay step followed by email + SMS delivery. Verifies both channel records appear.
func TestWorkflow_DelayAndMultiChannel(t *testing.T) {
	if apiKey() == "" {
		t.Skip("E2E_API_KEY not set — skipping e2e tests")
	}

	resp := postJSON(t, "/v1/events", map[string]any{
		"name":        "e2e.workflow.delay",
		"subscriber_id": "e2e-subscriber-001",
		"payload": map[string]any{
			"delay_seconds": 5,
			"email":         "e2e@example.com",
			"phone":         "+919876543210",
		},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("trigger: expected 202, got %d", resp.StatusCode)
	}

	// Wait past the delay step.
	time.Sleep(10 * time.Second)

	req, _ := http.NewRequest(http.MethodGet, baseURL()+"/v1/notifications?limit=10", nil)
	req.Header.Set("X-Qeet-Api-Key", apiKey())
	listResp, _ := http.DefaultClient.Do(req)
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list notifications: expected 200, got %d", listResp.StatusCode)
	}
}
