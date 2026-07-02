package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"

	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

type triggerEventRequest struct {
	Event        string         `json:"event"`
	SubscriberID string         `json:"subscriber_id"`
	Payload      map[string]any `json:"payload"`
}

const (
	// idempotencyTTL is how long a processed Idempotency-Key is remembered.
	idempotencyTTL = 24 * time.Hour
	// idempotencyPending marks a key that has been claimed but whose request
	// has not finished; a concurrent duplicate sees this and gets a 409.
	idempotencyPending = "__pending__"
)

// NewTriggerEvent returns the event intake handler. When the client supplies an
// `Idempotency-Key` header, duplicate requests (same tenant + key within 24h)
// are de-duplicated via Redis: the first request is processed and its response
// cached; retries replay that response (with `Idempotent-Replayed: true`) without
// re-queuing the event. Requests without the header behave as before. Redis
// errors fail open (the event is still queued), matching the rate limiter.
func NewTriggerEvent(js jetstream.JetStream, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := middleware.TenantFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"missing tenant context"}`, http.StatusUnauthorized)
			return
		}

		// Idempotency: claim the key before doing any work.
		var redisKey string
		if key := r.Header.Get("Idempotency-Key"); key != "" && rdb != nil {
			redisKey = fmt.Sprintf("idem:%s:%s", tenantID, key)
			claimed, err := rdb.SetNX(r.Context(), redisKey, idempotencyPending, idempotencyTTL).Result()
			if err == nil && !claimed {
				// Duplicate request: replay the cached response, or 409 if the
				// original is still in flight.
				prev, gerr := rdb.Get(r.Context(), redisKey).Result()
				if gerr == nil && prev == idempotencyPending {
					http.Error(w, `{"error":"a request with this Idempotency-Key is still being processed"}`, http.StatusConflict)
					return
				}
				if gerr == nil {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Idempotent-Replayed", "true")
					w.WriteHeader(http.StatusAccepted)
					w.Write([]byte(prev)) //nolint:errcheck
					return
				}
				// GET failed → fall through and process (fail open).
			}
			// SetNX error → redisKey left set; releaseIdem is a no-op-safe Del.
			if err != nil {
				redisKey = "" // fail open: don't try to cache/release on a broken Redis
			}
		}

		var req triggerEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			releaseIdem(r.Context(), rdb, redisKey)
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Event == "" {
			releaseIdem(r.Context(), rdb, redisKey)
			http.Error(w, `{"error":"event is required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.SubscriberID == "" {
			releaseIdem(r.Context(), rdb, redisKey)
			http.Error(w, `{"error":"subscriber_id is required"}`, http.StatusUnprocessableEntity)
			return
		}

		ev := engine.Event{
			TenantID:     tenantID,
			SubscriberID: req.SubscriberID,
			Event:        req.Event,
			Payload:      req.Payload,
		}
		data, _ := json.Marshal(ev)

		subject := messaging.EventSubject(tenantID)
		if _, err := js.Publish(context.Background(), subject, data); err != nil {
			releaseIdem(r.Context(), rdb, redisKey)
			http.Error(w, `{"error":"failed to queue event"}`, http.StatusInternalServerError)
			return
		}

		const body = `{"status":"accepted"}`
		// Cache the final response so retries replay it (best-effort).
		if redisKey != "" && rdb != nil {
			rdb.Set(r.Context(), redisKey, body, idempotencyTTL) //nolint:errcheck
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(body)) //nolint:errcheck
	}
}

// releaseIdem clears a claimed idempotency key when the request fails before
// producing a cacheable response, so the client can safely retry.
func releaseIdem(ctx context.Context, rdb *redis.Client, key string) {
	if key != "" && rdb != nil {
		rdb.Del(ctx, key) //nolint:errcheck
	}
}
