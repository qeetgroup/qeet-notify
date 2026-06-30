package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apimw "github.com/qeetgroup/qeet-notify/internal/api/middleware"
	"github.com/qeetgroup/qeet-notify/internal/preference"
)

// Unsubscribe processes one-click unsubscribe from a signed token in the URL.
// Used for List-Unsubscribe headers and hosted preference page links.
func Unsubscribe(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// token encodes tenantID:subscriberID:channel — validated by signed JWT in Step 10.
		// For now, accept explicit query params in dev.
		tenantID := r.URL.Query().Get("tenant_id")
		subscriberID := r.URL.Query().Get("subscriber_id")
		channel := r.URL.Query().Get("channel")
		if channel == "" {
			channel = "all"
		}

		if tenantID == "" || subscriberID == "" {
			http.Error(w, `{"error":"missing params"}`, http.StatusBadRequest)
			return
		}

		if err := preference.Unsubscribe(r.Context(), pool, tenantID, subscriberID, channel); err != nil {
			http.Error(w, `{"error":"unsubscribe failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"unsubscribed"}`)) //nolint:errcheck
	}
}

// GetPreferences returns a subscriber's channel+category opt-in matrix.
func GetPreferences(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		subscriberID := chi.URLParam(r, "subscriberID")

		rows, err := pool.Query(r.Context(),
			`SELECT channel, category, is_opted_in FROM preferences
			 WHERE tenant_id = $1 AND subscriber_id = $2`,
			tenantID, subscriberID,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type row struct {
			Channel    string `json:"channel"`
			Category   string `json:"category"`
			IsOptedIn  bool   `json:"is_opted_in"`
		}
		var prefs []row
		for rows.Next() {
			var p row
			rows.Scan(&p.Channel, &p.Category, &p.IsOptedIn) //nolint:errcheck
			prefs = append(prefs, p)
		}
		if prefs == nil {
			prefs = []row{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"preferences": prefs}) //nolint:errcheck
	}
}

// DeleteSubscriber hard-deletes PII (DPDP right to erasure).
func DeleteSubscriber(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		subscriberID := chi.URLParam(r, "subscriberID")

		if err := preference.EraseSubscriber(r.Context(), pool, tenantID, subscriberID); err != nil {
			http.Error(w, `{"error":"erasure failed"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
