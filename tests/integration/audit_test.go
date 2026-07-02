//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
)

func randUUID(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// TestAuditMiddleware verifies the Audit middleware writes an audit row for a
// mutating request, attributes the API-key actor, and skips GET and the event
// intake (Module 29).
func TestAuditMiddleware(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()
	ctx := context.Background()

	tenantID := randUUID(t)
	resourceID := randUUID(t)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE tenant_id = $1`, tenantID) })

	lookup := middleware.TenantLookup(func(context.Context, string) (string, string, bool, error) {
		return tenantID, "full", true, nil
	})
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }

	router := chi.NewRouter()
	router.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Auth(lookup))
		r.Use(middleware.Audit(pool))
		r.Post("/things/{id}", ok)
		r.Get("/things/{id}", ok)
		r.Post("/events", ok)
	})

	call := func(method, path string) {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("X-Qeet-Api-Key", "dummy-key")
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	call(http.MethodPost, "/v1/things/"+resourceID) // audited
	call(http.MethodGet, "/v1/things/"+resourceID)  // not audited (GET)
	call(http.MethodPost, "/v1/events")             // not audited (excluded)

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE tenant_id = $1`, tenantID).Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("audit rows = %d, want 1 (only the POST /things/{id})", count)
	}

	var action, resourceType, actorType, actorID string
	var gotResourceID *string
	if err := pool.QueryRow(ctx,
		`SELECT action, resource_type, resource_id::text, actor_type, actor_id
		 FROM audit_logs WHERE tenant_id = $1`, tenantID,
	).Scan(&action, &resourceType, &gotResourceID, &actorType, &actorID); err != nil {
		t.Fatalf("read audit row: %v", err)
	}
	if action != "POST /v1/things/{id}" {
		t.Errorf("action = %q, want %q", action, "POST /v1/things/{id}")
	}
	if resourceType != "things" {
		t.Errorf("resource_type = %q, want things", resourceType)
	}
	if gotResourceID == nil || *gotResourceID != resourceID {
		t.Errorf("resource_id = %v, want %s", gotResourceID, resourceID)
	}
	if actorType != "api_key" {
		t.Errorf("actor_type = %q, want api_key", actorType)
	}
	if len(actorID) == 0 || actorID[:7] != "apikey:" {
		t.Errorf("actor_id = %q, want apikey:* fingerprint", actorID)
	}
}
