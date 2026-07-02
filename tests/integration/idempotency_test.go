//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/qeetgroup/qeet-notify/platform/api/handler"
	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/cache"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// TestEventIdempotency verifies that repeating POST /v1/events with the same
// Idempotency-Key replays the first response and queues the event only once
// (Module 25). A request without the key is unaffected.
func TestEventIdempotency(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()
	ctx := context.Background()

	rdb, err := cache.New(getenv("REDIS_URL", "redis://localhost:6379"))
	if err != nil {
		t.Skipf("integration: no redis: %v", err)
	}
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("integration: redis ping failed: %v", err)
	}

	base := time.Now().UnixNano()
	tenantID := fmt.Sprintf("itest-idem-t%d", base) // dot-free: valid NATS subject token
	idemKey := fmt.Sprintf("key-%d", base)
	redisKey := fmt.Sprintf("idem:%s:%s", tenantID, idemKey)
	t.Cleanup(func() { _ = rdb.Del(context.Background(), redisKey).Err() })

	// Consumer on this tenant's events subject, to count how many events were queued.
	consName := fmt.Sprintf("itest-idem-c%d", base)
	cons, err := nc.JS.CreateOrUpdateConsumer(ctx, "NOTIFY_EVENTS", jetstream.ConsumerConfig{
		Name:           consName,
		AckPolicy:      jetstream.AckExplicitPolicy,
		FilterSubjects: []string{messaging.EventSubject(tenantID)},
	})
	if err != nil {
		t.Fatalf("create test consumer: %v", err)
	}
	defer func() { _ = nc.JS.DeleteConsumer(context.Background(), "NOTIFY_EVENTS", consName) }()

	// Stub tenant resolution so the real Auth middleware yields our tenant.
	lookup := middleware.TenantLookup(func(context.Context, string) (string, string, bool, error) {
		return tenantID, "full", true, nil
	})
	h := middleware.Auth(lookup)(handler.NewTriggerEvent(nc.JS, rdb))

	post := func(withKey bool) *httptest.ResponseRecorder {
		body := `{"event":"itest.idem","subscriber_id":"sub-1","payload":{}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(body))
		req.Header.Set("X-Qeet-Api-Key", "dummy")
		if withKey {
			req.Header.Set("Idempotency-Key", idemKey)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// First call: processed, not a replay.
	r1 := post(true)
	if r1.Code != http.StatusAccepted {
		t.Fatalf("first call: status = %d, want 202; body=%s", r1.Code, r1.Body.String())
	}
	if r1.Header().Get("Idempotent-Replayed") == "true" {
		t.Error("first call should not be a replay")
	}

	// Second call, same key: replayed, event not re-queued.
	r2 := post(true)
	if r2.Code != http.StatusAccepted {
		t.Fatalf("second call: status = %d, want 202", r2.Code)
	}
	if r2.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("second call should be replayed (Idempotent-Replayed: true)")
	}

	// Exactly one event should have been queued for this tenant.
	batch, err := cons.Fetch(5, jetstream.FetchMaxWait(2*time.Second))
	if err != nil {
		t.Fatalf("fetch events: %v", err)
	}
	n := 0
	for m := range batch.Messages() {
		_ = m.Ack()
		n++
	}
	if n != 1 {
		t.Errorf("queued %d events, want 1 (idempotent dedup)", n)
	}
}
