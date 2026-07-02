package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/domains/analytics"
	apimw "github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/database"
)

// DeliveryAnalytics returns aggregate delivery funnel totals for the authenticated tenant.
// Response: { queued, sent, delivered, failed, opened }
func DeliveryAnalytics(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)

		totals, err := analytics.QueryTotals(r.Context(), q, tenantID)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(totals) //nolint:errcheck
	}
}
