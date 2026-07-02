package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/database"
)

type auditLogRow struct {
	ID           string         `json:"id"`
	ActorType    string         `json:"actor_type"`
	ActorID      string         `json:"actor_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	OldValue     map[string]any `json:"old_value,omitempty"`
	NewValue     map[string]any `json:"new_value,omitempty"`
	IPAddress    *string        `json:"ip_address,omitempty"`
	OccurredAt   time.Time      `json:"occurred_at"`
}

// ListAuditLogs returns paginated audit trail entries for the authenticated tenant.
// Filters: action, resource_type, actor_id, from, to (RFC3339).
func ListAuditLogs(pool *pgxpool.Pool) http.HandlerFunc {
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

		baseWhere := `WHERE tenant_id = $1`
		args := []any{tenantID}

		if action := r.URL.Query().Get("action"); action != "" {
			args = append(args, action)
			baseWhere += ` AND action = $` + strconv.Itoa(len(args))
		}
		if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
			args = append(args, resourceType)
			baseWhere += ` AND resource_type = $` + strconv.Itoa(len(args))
		}
		if actorID := r.URL.Query().Get("actor_id"); actorID != "" {
			args = append(args, actorID)
			baseWhere += ` AND actor_id = $` + strconv.Itoa(len(args))
		}
		if from := r.URL.Query().Get("from"); from != "" {
			if t, err := time.Parse(time.RFC3339, from); err == nil {
				args = append(args, t)
				baseWhere += ` AND occurred_at >= $` + strconv.Itoa(len(args))
			}
		}
		if to := r.URL.Query().Get("to"); to != "" {
			if t, err := time.Parse(time.RFC3339, to); err == nil {
				args = append(args, t)
				baseWhere += ` AND occurred_at <= $` + strconv.Itoa(len(args))
			}
		}

		var total int64
		q.QueryRow(r.Context(), //nolint:errcheck
			`SELECT COUNT(*) FROM audit_logs `+baseWhere, args...,
		).Scan(&total)

		pageArgs := append(args, limit, offset) //nolint:gocritic
		query := `SELECT id, actor_type, actor_id, action, resource_type,
		                 resource_id::text, old_value, new_value, ip_address, occurred_at
		          FROM audit_logs ` + baseWhere +
			` ORDER BY occurred_at DESC LIMIT $` + strconv.Itoa(len(pageArgs)-1) +
			` OFFSET $` + strconv.Itoa(len(pageArgs))

		rows, err := q.Query(r.Context(), query, pageArgs...)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var logs []auditLogRow
		for rows.Next() {
			var row auditLogRow
			var oldVal, newVal []byte
			if err := rows.Scan(&row.ID, &row.ActorType, &row.ActorID, &row.Action,
				&row.ResourceType, &row.ResourceID, &oldVal, &newVal,
				&row.IPAddress, &row.OccurredAt); err != nil {
				continue
			}
			if oldVal != nil {
				json.Unmarshal(oldVal, &row.OldValue) //nolint:errcheck
			}
			if newVal != nil {
				json.Unmarshal(newVal, &row.NewValue) //nolint:errcheck
			}
			logs = append(logs, row)
		}
		if logs == nil {
			logs = []auditLogRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"logs": logs, "total": total}) //nolint:errcheck
	}
}
