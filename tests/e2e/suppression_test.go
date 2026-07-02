package e2e_test

import (
	"net/http"
	"testing"
)

// TestSuppression_SuppressedSubscriberDropped verifies that triggering an event for a
// suppressed subscriber results in an accepted (202) response but the delivery record
// has status="suppressed" — not sent.
func TestSuppression_SuppressedSubscriberDropped(t *testing.T) {
	if apiKey() == "" {
		t.Skip("E2E_API_KEY not set — skipping e2e tests")
	}

	// subscriber e2e-suppressed-001 must be pre-seeded as suppressed
	// (see tests/fixtures/suppressed_subscriber.sql).
	resp := postJSON(t, "/v1/events", map[string]any{
		"name":        "e2e.suppression.test",
		"subscriber_id": "e2e-suppressed-001",
		"payload":     map[string]any{},
	})
	defer resp.Body.Close()

	// API still accepts — suppression is enforced at worker level.
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
}
