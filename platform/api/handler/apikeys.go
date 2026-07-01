package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
)

type apiKeyRow struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	Scope     string     `json:"scope"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// ListAPIKeys returns all API keys for the tenant (plaintext never exposed after creation).
func ListAPIKeys(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())

		rows, err := pool.Query(r.Context(),
			`SELECT id, name, prefix, scope, created_at, revoked_at
			 FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC`,
			tenantID,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var keys []apiKeyRow
		for rows.Next() {
			var k apiKeyRow
			if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.Scope, &k.CreatedAt, &k.RevokedAt); err != nil {
				continue
			}
			keys = append(keys, k)
		}
		if keys == nil {
			keys = []apiKeyRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"api_keys": keys}) //nolint:errcheck
	}
}

// CreateAPIKey generates a new scoped API key. The raw key is returned only once.
func CreateAPIKey(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())

		var req struct {
			Name  string `json:"name"`
			Scope string `json:"scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.Scope == "" {
			req.Scope = "full"
		}

		rawKey, keyHash, prefix, err := generateAPIKey()
		if err != nil {
			http.Error(w, `{"error":"key generation failed"}`, http.StatusInternalServerError)
			return
		}

		var id string
		err = pool.QueryRow(r.Context(),
			`INSERT INTO api_keys (tenant_id, name, key_hash, prefix, scope)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			tenantID, req.Name, keyHash, prefix, req.Scope,
		).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"id":      id,
			"key":     rawKey,
			"prefix":  prefix,
			"scope":   req.Scope,
			"warning": "Store this key securely — it will not be shown again.",
		})
	}
}

// RevokeAPIKey marks an API key as revoked.
func RevokeAPIKey(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		id := chi.URLParam(r, "id")

		result, err := pool.Exec(r.Context(),
			`UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found or already revoked"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// generateAPIKey creates a cryptographically random key with prefix "qn_live_".
func generateAPIKey() (rawKey, keyHash, prefix string, err error) {
	buf := make([]byte, 24)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawKey = "qn_live_" + hex.EncodeToString(buf)
	sum := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(sum[:])
	prefix = rawKey[:16] // "qn_live_" + 8 hex chars
	return rawKey, keyHash, prefix, nil
}
