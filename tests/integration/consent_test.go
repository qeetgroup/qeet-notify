//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/qeetgroup/qeet-notify/platform/api/handler"
	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/database"
)

const consentEncKey = "test-enc-key"

// TestConsentLedgerAndExport verifies that updating preferences appends consent
// ledger rows (Modules 22/34) and that the DSR export endpoint returns the
// subscriber's decrypted PII, preferences, and consent history.
func TestConsentLedgerAndExport(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()
	ctx := context.Background()

	uniq := time.Now().UnixNano()
	var tenantID, subID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, api_key_hash, api_key_prefix)
		 VALUES ('itest', $1, $2, 'itest') RETURNING id`,
		fmt.Sprintf("itest-%d", uniq), fmt.Sprintf("hash-%d", uniq),
	).Scan(&tenantID); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID) })

	// subscribers is RLS-forced (Module 36): seed inside a tenant-scoped tx.
	if err := database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, q database.Querier) error {
		return q.QueryRow(ctx,
			`INSERT INTO subscribers (tenant_id, external_id, email_encrypted)
			 VALUES ($1, $2, pgp_sym_encrypt($3::text, $4)::text) RETURNING id`,
			tenantID, fmt.Sprintf("user-%d", uniq), "dsr@example.com", consentEncKey,
		).Scan(&subID)
	}); err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}

	lookup := middleware.TenantLookup(func(context.Context, string) (string, string, bool, error) {
		return tenantID, "full", true, nil
	})
	router := chi.NewRouter()
	router.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Auth(lookup))
		r.Use(middleware.TenantTx(pool))
		r.Put("/subscribers/{subscriberID}/preferences", handler.UpdatePreferences(pool))
		r.Get("/subscribers/{subscriberID}/data", handler.ExportSubscriberData(pool, consentEncKey))
	})

	// 1. Update two preferences → expect two consent-ledger rows.
	body := `{"preferences":[
		{"channel":"email","category":"marketing","is_opted_in":true},
		{"channel":"sms","category":"marketing","is_opted_in":false}
	]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/subscribers/"+subID+"/preferences", strings.NewReader(body))
	req.Header.Set("X-Qeet-Api-Key", "dummy-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update preferences: status %d, body %s", rec.Code, rec.Body.String())
	}

	var ledgerCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM consent_ledger WHERE tenant_id=$1 AND subscriber_id=$2`, tenantID, subID,
	).Scan(&ledgerCount); err != nil {
		t.Fatalf("count consent: %v", err)
	}
	if ledgerCount != 2 {
		t.Errorf("consent_ledger rows = %d, want 2", ledgerCount)
	}

	// 2. DSR export returns PII + preferences + consent history.
	req = httptest.NewRequest(http.MethodGet, "/v1/subscribers/"+subID+"/data", nil)
	req.Header.Set("X-Qeet-Api-Key", "dummy-key")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export: status %d, body %s", rec.Code, rec.Body.String())
	}

	var out struct {
		Subscriber     map[string]any   `json:"subscriber"`
		Preferences    []map[string]any `json:"preferences"`
		ConsentHistory []map[string]any `json:"consent_history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if got, _ := out.Subscriber["email"].(string); got != "dsr@example.com" {
		t.Errorf("export email = %q, want dsr@example.com (decrypted PII)", got)
	}
	if len(out.Preferences) != 2 {
		t.Errorf("export preferences = %d, want 2", len(out.Preferences))
	}
	if len(out.ConsentHistory) != 2 {
		t.Errorf("export consent_history = %d, want 2", len(out.ConsentHistory))
	}
}
