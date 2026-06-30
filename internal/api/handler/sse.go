package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// NotificationStream serves a Server-Sent Events stream for a subscriber.
// URL params: tenantID, subscriberID (extracted from short-lived subscriber token by auth middleware).
func NotificationStream(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantID")
		subscriberID := chi.URLParam(r, "subscriberID")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		channel := fmt.Sprintf("notify:inapp:%s:%s", tenantID, subscriberID)
		sub := rdb.Subscribe(r.Context(), channel)
		defer sub.Close()

		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case msg := <-sub.Channel():
				fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
				flusher.Flush()
			}
		}
	}
}

// NotificationFeed returns paginated in-app notifications for a subscriber.
func NotificationFeed(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantID")
		subscriberID := chi.URLParam(r, "subscriberID")

		rows, err := pool.Query(r.Context(),
			`SELECT id, content, is_read, created_at
			 FROM notifications
			 WHERE tenant_id = $1 AND subscriber_id = $2 AND channel = 'inapp'
			 ORDER BY created_at DESC LIMIT 50`,
			tenantID, subscriberID,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type item struct {
			ID        string    `json:"id"`
			Content   any       `json:"content"`
			IsRead    bool      `json:"is_read"`
			CreatedAt time.Time `json:"created_at"`
		}
		var result []item
		for rows.Next() {
			var n item
			var contentRaw []byte
			if err := rows.Scan(&n.ID, &contentRaw, &n.IsRead, &n.CreatedAt); err != nil {
				continue
			}
			json.Unmarshal(contentRaw, &n.Content) //nolint:errcheck
			result = append(result, n)
		}
		if result == nil {
			result = []item{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"notifications": result}) //nolint:errcheck
	}
}

// MarkNotificationRead marks an in-app notification as read.
func MarkNotificationRead(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifID := chi.URLParam(r, "notifID")
		tenantID := chi.URLParam(r, "tenantID")

		if _, err := pool.Exec(r.Context(),
			`UPDATE notifications SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2 AND channel = 'inapp'`,
			notifID, tenantID,
		); err != nil {
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	}
}
