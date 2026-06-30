package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/qeetgroup/qeet-notify/internal/api/middleware"
	platformnats "github.com/qeetgroup/qeet-notify/internal/platform/nats"
	"github.com/qeetgroup/qeet-notify/internal/workflow"
)

type triggerEventRequest struct {
	Event        string         `json:"event"`
	SubscriberID string         `json:"subscriber_id"`
	Payload      map[string]any `json:"payload"`
}

// NewTriggerEvent returns the event intake handler wired to the NATS JetStream client.
func NewTriggerEvent(js jetstream.JetStream) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := middleware.TenantFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"missing tenant context"}`, http.StatusUnauthorized)
			return
		}

		var req triggerEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Event == "" {
			http.Error(w, `{"error":"event is required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.SubscriberID == "" {
			http.Error(w, `{"error":"subscriber_id is required"}`, http.StatusUnprocessableEntity)
			return
		}

		ev := workflow.Event{
			TenantID:     tenantID,
			SubscriberID: req.SubscriberID,
			Event:        req.Event,
			Payload:      req.Payload,
		}
		data, _ := json.Marshal(ev)

		subject := platformnats.EventSubject(tenantID)
		if _, err := js.Publish(context.Background(), subject, data); err != nil {
			http.Error(w, `{"error":"failed to queue event"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`)) //nolint:errcheck
	}
}
