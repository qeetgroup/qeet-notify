package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
)

type notificationRow struct {
	ID                 string         `json:"id"`
	WorkflowRunID      *string        `json:"workflow_run_id,omitempty"`
	SubscriberID       string         `json:"subscriber_id"`
	Channel            string         `json:"channel"`
	TemplateID         *string        `json:"template_id,omitempty"`
	Status             string         `json:"status"`
	Provider           *string        `json:"provider,omitempty"`
	ProviderMessageID  *string        `json:"provider_message_id,omitempty"`
	Content            map[string]any `json:"content"`
	IsRead             bool           `json:"is_read"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// ListNotifications returns paginated notifications for the authenticated tenant.
func ListNotifications(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())

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
		filterArgs := []any{tenantID}

		if status := r.URL.Query().Get("status"); status != "" {
			filterArgs = append(filterArgs, status)
			baseWhere += ` AND status = $` + strconv.Itoa(len(filterArgs))
		}
		if channel := r.URL.Query().Get("channel"); channel != "" {
			filterArgs = append(filterArgs, channel)
			baseWhere += ` AND channel = $` + strconv.Itoa(len(filterArgs))
		}
		if subscriberID := r.URL.Query().Get("subscriber_id"); subscriberID != "" {
			filterArgs = append(filterArgs, subscriberID)
			baseWhere += ` AND subscriber_id = $` + strconv.Itoa(len(filterArgs))
		}

		var total int64
		pool.QueryRow(r.Context(), //nolint:errcheck
			`SELECT COUNT(*) FROM notifications `+baseWhere, filterArgs...,
		).Scan(&total)

		pageArgs := append(filterArgs, limit, offset) //nolint:gocritic
		query := `SELECT id, workflow_run_id, subscriber_id, channel, template_id, status,
		                 provider, provider_message_id, content, is_read, created_at, updated_at
		          FROM notifications ` + baseWhere +
			` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(pageArgs)-1) +
			` OFFSET $` + strconv.Itoa(len(pageArgs))

		rows, err := pool.Query(r.Context(), query, pageArgs...)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var notifications []notificationRow
		for rows.Next() {
			var n notificationRow
			var content []byte
			if err := rows.Scan(&n.ID, &n.WorkflowRunID, &n.SubscriberID, &n.Channel, &n.TemplateID,
				&n.Status, &n.Provider, &n.ProviderMessageID, &content, &n.IsRead,
				&n.CreatedAt, &n.UpdatedAt); err != nil {
				continue
			}
			json.Unmarshal(content, &n.Content) //nolint:errcheck
			notifications = append(notifications, n)
		}
		if notifications == nil {
			notifications = []notificationRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"notifications": notifications, "total": total}) //nolint:errcheck
	}
}

// GetNotification returns a single notification with its delivery event history.
func GetNotification(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		id := chi.URLParam(r, "id")

		var n notificationRow
		var content []byte
		err := pool.QueryRow(r.Context(),
			`SELECT id, workflow_run_id, subscriber_id, channel, template_id, status,
			        provider, provider_message_id, content, is_read, created_at, updated_at
			 FROM notifications WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&n.ID, &n.WorkflowRunID, &n.SubscriberID, &n.Channel, &n.TemplateID,
			&n.Status, &n.Provider, &n.ProviderMessageID, &content, &n.IsRead,
			&n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.Unmarshal(content, &n.Content) //nolint:errcheck

		// Fetch delivery events
		type deliveryEvent struct {
			ID         string    `json:"id"`
			EventType  string    `json:"event_type"`
			Provider   string    `json:"provider"`
			OccurredAt time.Time `json:"occurred_at"`
		}
		evRows, err := pool.Query(r.Context(),
			`SELECT id, event_type, provider, occurred_at
			 FROM delivery_events WHERE notification_id = $1 ORDER BY occurred_at`,
			id,
		)
		var events []deliveryEvent
		if err == nil {
			defer evRows.Close()
			for evRows.Next() {
				var ev deliveryEvent
				if err := evRows.Scan(&ev.ID, &ev.EventType, &ev.Provider, &ev.OccurredAt); err == nil {
					events = append(events, ev)
				}
			}
		}
		if events == nil {
			events = []deliveryEvent{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"notification":    n,
			"delivery_events": events,
		})
	}
}
