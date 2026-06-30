package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/internal/analytics"
	apimw "github.com/qeetgroup/qeet-notify/internal/api/middleware"
)

// DeliveryAnalytics returns 30-day delivery stats for the authenticated tenant.
func DeliveryAnalytics(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())

		stats, err := analytics.QueryDelivery(r.Context(), pool, tenantID)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"stats": stats}) //nolint:errcheck
	}
}
