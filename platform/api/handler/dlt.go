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

type dltTemplateRow struct {
	ID            string         `json:"id"`
	Carrier       string         `json:"carrier"`
	Channel       string         `json:"channel"`
	TemplateIDExt string         `json:"template_id_ext"`
	TemplateName  string         `json:"template_name"`
	PeID          *string        `json:"pe_id,omitempty"`
	SenderID      *string        `json:"sender_id,omitempty"`
	Category      string         `json:"category"`
	BodyRegex     string         `json:"body_regex"`
	Status        string         `json:"status"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// ListDLTTemplates returns all DLT templates for the authenticated tenant.
func ListDLTTemplates(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)

		channel := r.URL.Query().Get("channel")
		carrier := r.URL.Query().Get("carrier")
		status := r.URL.Query().Get("status")

		query := `SELECT id, carrier, channel, template_id_ext, template_name, pe_id, sender_id,
		                 category, body_regex, status, metadata, created_at, updated_at
		          FROM dlt_templates WHERE tenant_id = $1`
		args := []any{tenantID}
		if channel != "" {
			args = append(args, channel)
			query += ` AND channel = $` + itoa(len(args))
		}
		if carrier != "" {
			args = append(args, carrier)
			query += ` AND carrier = $` + itoa(len(args))
		}
		if status != "" {
			args = append(args, status)
			query += ` AND status = $` + itoa(len(args))
		}
		query += ` ORDER BY created_at DESC`

		rows, err := q.Query(r.Context(), query, args...)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var templates []dltTemplateRow
		for rows.Next() {
			var t dltTemplateRow
			var meta []byte
			if err := rows.Scan(&t.ID, &t.Carrier, &t.Channel, &t.TemplateIDExt, &t.TemplateName,
				&t.PeID, &t.SenderID, &t.Category, &t.BodyRegex, &t.Status, &meta,
				&t.CreatedAt, &t.UpdatedAt); err != nil {
				continue
			}
			json.Unmarshal(meta, &t.Metadata) //nolint:errcheck
			templates = append(templates, t)
		}
		if templates == nil {
			templates = []dltTemplateRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"dlt_templates": templates}) //nolint:errcheck
	}
}

// RegisterDLTTemplate creates a new DLT template registration.
func RegisterDLTTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)

		var req struct {
			Carrier       string         `json:"carrier"`
			Channel       string         `json:"channel"`
			TemplateIDExt string         `json:"template_id_ext"`
			TemplateName  string         `json:"template_name"`
			PeID          *string        `json:"pe_id"`
			SenderID      *string        `json:"sender_id"`
			Category      string         `json:"category"`
			BodyRegex     string         `json:"body_regex"`
			Metadata      map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Carrier == "" || req.TemplateIDExt == "" || req.TemplateName == "" || req.BodyRegex == "" {
			http.Error(w, `{"error":"carrier, template_id_ext, template_name, and body_regex are required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.Channel == "" {
			req.Channel = "sms"
		}
		if req.Category == "" {
			req.Category = "transactional"
		}
		if req.Metadata == nil {
			req.Metadata = map[string]any{}
		}
		meta, _ := json.Marshal(req.Metadata)

		var id string
		err := q.QueryRow(r.Context(),
			`INSERT INTO dlt_templates
			 (tenant_id, carrier, channel, template_id_ext, template_name, pe_id, sender_id, category, body_regex, metadata)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
			tenantID, req.Carrier, req.Channel, req.TemplateIDExt, req.TemplateName,
			req.PeID, req.SenderID, req.Category, req.BodyRegex, meta,
		).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "pending"}) //nolint:errcheck
	}
}

// UpdateDLTTemplate updates a DLT template's fields.
func UpdateDLTTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		var req struct {
			Status   *string `json:"status"`
			PeID     *string `json:"pe_id"`
			SenderID *string `json:"sender_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.Status != nil {
			result, err := q.Exec(r.Context(),
				`UPDATE dlt_templates SET status=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3`,
				*req.Status, id, tenantID,
			)
			if err != nil || result.RowsAffected() == 0 {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
		}
		if req.PeID != nil {
			q.Exec(r.Context(), //nolint:errcheck
				`UPDATE dlt_templates SET pe_id=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3`,
				*req.PeID, id, tenantID,
			)
		}
		if req.SenderID != nil {
			q.Exec(r.Context(), //nolint:errcheck
				`UPDATE dlt_templates SET sender_id=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3`,
				*req.SenderID, id, tenantID,
			)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
	}
}

// DeleteDLTTemplate removes a DLT template.
func DeleteDLTTemplate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		result, err := q.Exec(r.Context(),
			`DELETE FROM dlt_templates WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// itoa converts int to string for query building.
func itoa(n int) string {
	return strconv.Itoa(n)
}
