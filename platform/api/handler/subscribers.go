package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/domains/subscribers/consent"
	"github.com/qeetgroup/qeet-notify/domains/subscribers/preferences"
	apimw "github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/database"
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

		// Public endpoint (no request tx from middleware): run the writes in a
		// tenant-scoped tx so row-level security applies.
		err := database.RunInTenant(r.Context(), pool, tenantID, func(ctx context.Context, q database.Querier) error {
			if uerr := preferences.Unsubscribe(ctx, q, tenantID, subscriberID, channel); uerr != nil {
				return uerr
			}
			// Append to the DPDP consent ledger (best-effort).
			_ = consent.Record(ctx, q, consent.Entry{
				TenantID: tenantID, SubscriberID: subscriberID,
				Channel: channel, Category: "all", OptedIn: false,
				Source: "unsubscribe_link", Actor: "subscriber", IP: apimw.ClientIP(r),
			})
			return nil
		})
		if err != nil {
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
		q := database.FromContext(r.Context(), pool)
		subscriberID := chi.URLParam(r, "subscriberID")

		rows, err := q.Query(r.Context(),
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
		q := database.FromContext(r.Context(), pool)
		subscriberID := chi.URLParam(r, "subscriberID")

		if err := preferences.EraseSubscriber(r.Context(), q, tenantID, subscriberID); err != nil {
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
		q := database.FromContext(r.Context(), pool)

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
			err := q.QueryRow(r.Context(),
				`SELECT pgp_sym_encrypt($1, $2)::text`, req.Email, encKey,
			).Scan(&enc)
			if err == nil {
				emailEnc = &enc
			}
		}
		if req.Phone != "" {
			var enc string
			err := q.QueryRow(r.Context(),
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
		err := q.QueryRow(r.Context(),
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
		q := database.FromContext(r.Context(), pool)

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
		q.QueryRow(r.Context(), //nolint:errcheck
			`SELECT COUNT(*) FROM subscribers WHERE tenant_id = $1 AND is_deleted = FALSE`,
			tenantID,
		).Scan(&total)

		rows, err := q.Query(r.Context(),
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
		q := database.FromContext(r.Context(), pool)
		subscriberID := chi.URLParam(r, "subscriberID")

		var s subscriberRow
		var meta []byte
		err := q.QueryRow(r.Context(),
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

// ExportSubscriberData serves a DPDP Data-Subject-Rights access/export request:
// all personal data held for one subscriber — decrypted profile PII, the
// current preference matrix, and the full consent history. Requires an API key
// (the tenant is the data fiduciary for its own subscribers).
func ExportSubscriberData(pool *pgxpool.Pool, encKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		subscriberID := chi.URLParam(r, "subscriberID")

		var (
			id, externalID, locale, tz string
			email, phone, waID         *string
			meta                       []byte
			isDeleted                  bool
			createdAt                  time.Time
		)
		err := q.QueryRow(r.Context(),
			`SELECT id, external_id,
			        notify_decrypt(email_encrypted, $3), notify_decrypt(phone_encrypted, $3), whatsapp_id,
			        locale, timezone, metadata, is_deleted, created_at
			 FROM subscribers WHERE id = $1 AND tenant_id = $2`,
			subscriberID, tenantID, encKey,
		).Scan(&id, &externalID, &email, &phone, &waID, &locale, &tz, &meta, &isDeleted, &createdAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var metaMap map[string]any
		json.Unmarshal(meta, &metaMap) //nolint:errcheck

		type prefRow struct {
			Channel   string `json:"channel"`
			Category  string `json:"category"`
			IsOptedIn bool   `json:"is_opted_in"`
		}
		prefs := []prefRow{}
		rows, err := q.Query(r.Context(),
			`SELECT channel, category, is_opted_in FROM preferences
			 WHERE tenant_id = $1 AND subscriber_id = $2`,
			tenantID, subscriberID,
		)
		if err == nil {
			for rows.Next() {
				var p prefRow
				rows.Scan(&p.Channel, &p.Category, &p.IsOptedIn) //nolint:errcheck
				prefs = append(prefs, p)
			}
			rows.Close()
		}

		history, err := consent.History(r.Context(), q, tenantID, subscriberID)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"subscriber": map[string]any{
				"id":          id,
				"external_id": externalID,
				"email":       email,
				"phone":       phone,
				"whatsapp_id": waID,
				"locale":      locale,
				"timezone":    tz,
				"metadata":    metaMap,
				"is_deleted":  isDeleted,
				"created_at":  createdAt,
			},
			"preferences":     prefs,
			"consent_history": history,
		})
	}
}

// UpdateSubscriber updates locale, timezone, or metadata.
func UpdateSubscriber(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := apimw.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
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
		err := q.QueryRow(r.Context(),
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

		_, err = q.Exec(r.Context(),
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
		q := database.FromContext(r.Context(), pool)
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

		_, actorID := apimw.ActorFromContext(r.Context())
		ip := apimw.ClientIP(r)

		for _, p := range req.Preferences {
			if p.Channel == "" {
				continue
			}
			cat := p.Category
			if cat == "" {
				cat = "all"
			}
			_, err := q.Exec(r.Context(),
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
			// Append to the DPDP consent ledger (best-effort).
			_ = consent.Record(r.Context(), q, consent.Entry{
				TenantID: tenantID, SubscriberID: subscriberID,
				Channel: p.Channel, Category: cat, OptedIn: p.IsOptedIn,
				Source: "api", Actor: actorID, IP: ip,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"}) //nolint:errcheck
	}
}
