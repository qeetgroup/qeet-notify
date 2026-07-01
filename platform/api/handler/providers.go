package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
)

type providerRow struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Provider  string    `json:"provider"`
	Priority  int       `json:"priority"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListProviders returns all provider configs for the tenant (credentials masked).
func ListProviders(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())

		rows, err := pool.Query(r.Context(),
			`SELECT id, channel, provider, priority, is_active, created_at, updated_at
			 FROM provider_configs WHERE tenant_id = $1 ORDER BY channel, priority`,
			tenantID,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var providers []providerRow
		for rows.Next() {
			var p providerRow
			if err := rows.Scan(&p.ID, &p.Channel, &p.Provider, &p.Priority, &p.IsActive,
				&p.CreatedAt, &p.UpdatedAt); err != nil {
				continue
			}
			providers = append(providers, p)
		}
		if providers == nil {
			providers = []providerRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"providers": providers}) //nolint:errcheck
	}
}

// CreateProvider stores a new provider config with encrypted credentials.
func CreateProvider(pool *pgxpool.Pool, encKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())

		var req struct {
			Channel  string         `json:"channel"`
			Provider string         `json:"provider"`
			Priority int            `json:"priority"`
			Config   map[string]any `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Channel == "" || req.Provider == "" || req.Config == nil {
			http.Error(w, `{"error":"channel, provider, and config are required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.Priority == 0 {
			req.Priority = 1
		}
		configJSON, _ := json.Marshal(req.Config)

		var id string
		err := pool.QueryRow(r.Context(),
			`INSERT INTO provider_configs (tenant_id, channel, provider, priority, config_encrypted)
			 VALUES ($1, $2, $3, $4, pgp_sym_encrypt($5::text, $6)) RETURNING id`,
			tenantID, req.Channel, req.Provider, req.Priority, string(configJSON), encKey,
		).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
	}
}

// UpdateProvider updates provider priority or active state.
func UpdateProvider(pool *pgxpool.Pool, encKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		id := chi.URLParam(r, "id")

		var req struct {
			Priority *int           `json:"priority"`
			IsActive *bool          `json:"is_active"`
			Config   map[string]any `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.Config != nil {
			configJSON, _ := json.Marshal(req.Config)
			_, err := pool.Exec(r.Context(),
				`UPDATE provider_configs
				 SET config_encrypted = pgp_sym_encrypt($1::text, $2), updated_at = NOW()
				 WHERE id = $3 AND tenant_id = $4`,
				string(configJSON), encKey, id, tenantID,
			)
			if err != nil {
				http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
				return
			}
		}
		if req.Priority != nil {
			pool.Exec(r.Context(), //nolint:errcheck
				`UPDATE provider_configs SET priority=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3`,
				*req.Priority, id, tenantID,
			)
		}
		if req.IsActive != nil {
			pool.Exec(r.Context(), //nolint:errcheck
				`UPDATE provider_configs SET is_active=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3`,
				*req.IsActive, id, tenantID,
			)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
	}
}

// DeleteProvider removes a provider config.
func DeleteProvider(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		id := chi.URLParam(r, "id")

		result, err := pool.Exec(r.Context(),
			`DELETE FROM provider_configs WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
