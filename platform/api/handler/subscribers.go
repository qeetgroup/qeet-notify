package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/domains/subscribers/preferences"
	apimw "github.com/qeetgroup/qeet-notify/platform/api/middleware"
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

		if err := preferences.Unsubscribe(r.Context(), pool, tenantID, subscriberID, channel); err != nil {
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

		if err := preferences.EraseSubscriber(r.Context(), pool, tenantID, subscriberID); err != nil {
			http.Error(w, `{"error":"erasure failed"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type subscriberRow struct {
	ID         string         `json:"id"`
	ExternalID string         `json:"external_id"`
	Locale     string         `json:"locale"`
	Timezone   string         `json:"timezone"`
	Metadata   map[string]any `json:"metadata"`
	IsDeleted  bool           `json:"is_deleted"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// CreateSubscriber creates a new subscriber, PGP-encrypting PII.
func CreateSubscriber(pool *pgxpool.Pool, encKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())

		var req struct {
			ExternalID  string         `json:"external_id"`
			Email       string         `json:"email"`
			Phone       string         `json:"phone"`
			WhatsAppID  string         `json:"whatsapp_id"`
			Locale      string         `json:"locale"`
			Timezone    string         `json:"timezone"`
			Metadata    map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.ExternalID == "" {
			http.Error(w, `{"error":"external_id is required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.Locale == "" {
			req.Locale = "en"
		}
		if req.Timezone == "" {
			req.Timezone = "UTC"
		}
		if req.Metadata == nil {
			req.Metadata = map[string]any{}
		}
		meta, _ := json.Marshal(req.Metadata)

		var emailEnc, phoneEnc *string
		if req.Email != "" {
			var enc string
			err := pool.QueryRow(r.Context(),
				`SELECT pgp_sym_encrypt($1, $2)::text`, req.Email, encKey,
			).Scan(&enc)
			if err == nil {
				emailEnc = &enc
			}
		}
		if req.Phone != "" {
			var enc string
			err := pool.QueryRow(r.Context(),
				`SELECT pgp_sym_encrypt($1, $2)::text`, req.Phone, encKey,
			).Scan(&enc)
			if err == nil {
				phoneEnc = &enc
			}
		}

		var waID *string
		if req.WhatsAppID != "" {
			waID = &req.WhatsAppID
		}

		var id string
		err := pool.QueryRow(r.Context(),
			`INSERT INTO subscribers (tenant_id, external_id, email_encrypted, phone_encrypted, whatsapp_id, locale, timezone, metadata)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
			tenantID, req.ExternalID, emailEnc, phoneEnc, waID, req.Locale, req.Timezone, meta,
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

// ListSubscribers returns paginated subscribers for the tenant.
func ListSubscribers(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())

		limit := 50
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		var total int64
		pool.QueryRow(r.Context(), //nolint:errcheck
			`SELECT COUNT(*) FROM subscribers WHERE tenant_id = $1 AND is_deleted = FALSE`,
			tenantID,
		).Scan(&total)

		rows, err := pool.Query(r.Context(),
			`SELECT id, external_id, locale, timezone, metadata, is_deleted, created_at, updated_at
			 FROM subscribers
			 WHERE tenant_id = $1 AND is_deleted = FALSE
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			tenantID, limit, offset,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var subs []subscriberRow
		for rows.Next() {
			var s subscriberRow
			var meta []byte
			if err := rows.Scan(&s.ID, &s.ExternalID, &s.Locale, &s.Timezone, &meta,
				&s.IsDeleted, &s.CreatedAt, &s.UpdatedAt); err != nil {
				continue
			}
			json.Unmarshal(meta, &s.Metadata) //nolint:errcheck
			subs = append(subs, s)
		}
		if subs == nil {
			subs = []subscriberRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"subscribers": subs, "total": total}) //nolint:errcheck
	}
}

// GetSubscriber returns a single subscriber by ID.
func GetSubscriber(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		subscriberID := chi.URLParam(r, "subscriberID")

		var s subscriberRow
		var meta []byte
		err := pool.QueryRow(r.Context(),
			`SELECT id, external_id, locale, timezone, metadata, is_deleted, created_at, updated_at
			 FROM subscribers WHERE id = $1 AND tenant_id = $2`,
			subscriberID, tenantID,
		).Scan(&s.ID, &s.ExternalID, &s.Locale, &s.Timezone, &meta,
			&s.IsDeleted, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.Unmarshal(meta, &s.Metadata) //nolint:errcheck

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s) //nolint:errcheck
	}
}

// UpdateSubscriber updates locale, timezone, or metadata.
func UpdateSubscriber(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		subscriberID := chi.URLParam(r, "subscriberID")

		var req struct {
			Locale   *string        `json:"locale"`
			Timezone *string        `json:"timezone"`
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		var cur subscriberRow
		var curMeta []byte
		err := pool.QueryRow(r.Context(),
			`SELECT id, external_id, locale, timezone, metadata, is_deleted, created_at, updated_at
			 FROM subscribers WHERE id=$1 AND tenant_id=$2`,
			subscriberID, tenantID,
		).Scan(&cur.ID, &cur.ExternalID, &cur.Locale, &cur.Timezone, &curMeta,
			&cur.IsDeleted, &cur.CreatedAt, &cur.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.Unmarshal(curMeta, &cur.Metadata) //nolint:errcheck

		if req.Locale != nil {
			cur.Locale = *req.Locale
		}
		if req.Timezone != nil {
			cur.Timezone = *req.Timezone
		}
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				cur.Metadata[k] = v
			}
		}
		meta, _ := json.Marshal(cur.Metadata)

		_, err = pool.Exec(r.Context(),
			`UPDATE subscribers SET locale=$1, timezone=$2, metadata=$3, updated_at=NOW()
			 WHERE id=$4 AND tenant_id=$5`,
			cur.Locale, cur.Timezone, meta, subscriberID, tenantID,
		)
		if err != nil {
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": subscriberID}) //nolint:errcheck
	}
}

// UpdatePreferences sets opt-in/out for a channel+category pair.
func UpdatePreferences(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		subscriberID := chi.URLParam(r, "subscriberID")

		var req struct {
			Preferences []struct {
				Channel   string `json:"channel"`
				Category  string `json:"category"`
				IsOptedIn bool   `json:"is_opted_in"`
			} `json:"preferences"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		for _, p := range req.Preferences {
			if p.Channel == "" {
				continue
			}
			cat := p.Category
			if cat == "" {
				cat = "all"
			}
			_, err := pool.Exec(r.Context(),
				`INSERT INTO preferences (tenant_id, subscriber_id, channel, category, is_opted_in)
				 VALUES ($1, $2, $3, $4, $5)
				 ON CONFLICT (tenant_id, subscriber_id, channel, category)
				 DO UPDATE SET is_opted_in=$5, updated_at=NOW()`,
				tenantID, subscriberID, p.Channel, cat, p.IsOptedIn,
			)
			if err != nil {
				http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"}) //nolint:errcheck
	}
}
