package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/database"
)

type templateRow struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Channel   string         `json:"channel"`
	Locale    string         `json:"locale"`
	Subject   *string        `json:"subject,omitempty"`
	Body      string         `json:"body"`
	Metadata  map[string]any `json:"metadata"`
	IsActive  bool           `json:"is_active"`
	Version   int            `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ListTemplates returns all templates for the authenticated tenant.
func ListTemplates(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
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
		channel := r.URL.Query().Get("channel")

		query := `SELECT id, name, channel, locale, subject, body, metadata, is_active,
		                 COALESCE((metadata->>'version')::int, 1), created_at, updated_at
		          FROM templates
		          WHERE tenant_id = $1`
		args := []any{tenantID}
		if channel != "" {
			query += ` AND channel = $2`
			args = append(args, channel)
			query += ` ORDER BY name LIMIT $3 OFFSET $4`
			args = append(args, limit, offset)
		} else {
			query += ` ORDER BY name LIMIT $2 OFFSET $3`
			args = append(args, limit, offset)
		}

		rows, err := q.Query(r.Context(), query, args...)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var templates []templateRow
		for rows.Next() {
			var t templateRow
			var meta []byte
			if err := rows.Scan(&t.ID, &t.Name, &t.Channel, &t.Locale, &t.Subject,
				&t.Body, &meta, &t.IsActive, &t.Version, &t.CreatedAt, &t.UpdatedAt); err != nil {
				continue
			}
			json.Unmarshal(meta, &t.Metadata) //nolint:errcheck
			templates = append(templates, t)
		}
		if templates == nil {
			templates = []templateRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"templates": templates, "total": len(templates)}) //nolint:errcheck
	}
}

// GetTemplate returns a single template by ID.
func GetTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		var t templateRow
		var meta []byte
		err := q.QueryRow(r.Context(),
			`SELECT id, name, channel, locale, subject, body, metadata, is_active,
			        COALESCE((metadata->>'version')::int, 1), created_at, updated_at
			 FROM templates WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&t.ID, &t.Name, &t.Channel, &t.Locale, &t.Subject,
			&t.Body, &meta, &t.IsActive, &t.Version, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.Unmarshal(meta, &t.Metadata) //nolint:errcheck

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(t) //nolint:errcheck
	}
}

// CreateTemplate creates a new template.
func CreateTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)

		var req struct {
			Name     string         `json:"name"`
			Channel  string         `json:"channel"`
			Locale   string         `json:"locale"`
			Subject  *string        `json:"subject"`
			Body     string         `json:"body"`
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Channel == "" || req.Body == "" {
			http.Error(w, `{"error":"name, channel, and body are required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.Locale == "" {
			req.Locale = "en"
		}
		if req.Metadata == nil {
			req.Metadata = map[string]any{}
		}
		req.Metadata["version"] = 1
		meta, _ := json.Marshal(req.Metadata)

		var id string
		err := q.QueryRow(r.Context(),
			`INSERT INTO templates (tenant_id, name, channel, locale, subject, body, metadata)
			 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			tenantID, req.Name, req.Channel, req.Locale, req.Subject, req.Body, meta,
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

// UpdateTemplate updates a template's body, subject, and metadata.
func UpdateTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		var req struct {
			Name     *string        `json:"name"`
			Subject  *string        `json:"subject"`
			Body     *string        `json:"body"`
			Locale   *string        `json:"locale"`
			Metadata map[string]any `json:"metadata"`
			IsActive *bool          `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Fetch current to merge
		var cur templateRow
		var curMeta []byte
		err := q.QueryRow(r.Context(),
			`SELECT id, name, channel, locale, subject, body, metadata, is_active,
			        COALESCE((metadata->>'version')::int, 1), created_at, updated_at
			 FROM templates WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&cur.ID, &cur.Name, &cur.Channel, &cur.Locale, &cur.Subject,
			&cur.Body, &curMeta, &cur.IsActive, &cur.Version, &cur.CreatedAt, &cur.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.Unmarshal(curMeta, &cur.Metadata) //nolint:errcheck

		if req.Name != nil {
			cur.Name = *req.Name
		}
		if req.Subject != nil {
			cur.Subject = req.Subject
		}
		if req.Body != nil {
			cur.Body = *req.Body
		}
		if req.Locale != nil {
			cur.Locale = *req.Locale
		}
		if req.IsActive != nil {
			cur.IsActive = *req.IsActive
		}
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				cur.Metadata[k] = v
			}
		}
		// Bump version on content change
		newVersion := cur.Version + 1
		cur.Metadata["version"] = newVersion
		meta, _ := json.Marshal(cur.Metadata)

		_, err = q.Exec(r.Context(),
			`UPDATE templates SET name=$1, subject=$2, body=$3, locale=$4, metadata=$5, is_active=$6, updated_at=NOW()
			 WHERE id=$7 AND tenant_id=$8`,
			cur.Name, cur.Subject, cur.Body, cur.Locale, meta, cur.IsActive, id, tenantID,
		)
		if err != nil {
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "version": newVersion}) //nolint:errcheck
	}
}

// DeleteTemplate soft-deletes a template (sets is_active=false).
func DeleteTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		result, err := q.Exec(r.Context(),
			`UPDATE templates SET is_active = FALSE, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// PublishTemplate bumps the published version marker in metadata.
func PublishTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		var curMeta []byte
		var version int
		err := q.QueryRow(r.Context(),
			`SELECT metadata, COALESCE((metadata->>'version')::int, 1) FROM templates WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		).Scan(&curMeta, &version)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		var meta map[string]any
		json.Unmarshal(curMeta, &meta) //nolint:errcheck
		if meta == nil {
			meta = map[string]any{}
		}
		meta["published_version"] = version
		meta["published_at"] = time.Now().UTC().Format(time.RFC3339)
		newMeta, _ := json.Marshal(meta)

		_, err = q.Exec(r.Context(),
			`UPDATE templates SET metadata=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3`,
			newMeta, id, tenantID,
		)
		if err != nil {
			http.Error(w, `{"error":"publish failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "published_version": version}) //nolint:errcheck
	}
}
