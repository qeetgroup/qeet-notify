//go:build integration

// White-box integration test for the SMS worker's NDNC/DND scrub (Module 32)
// and its interaction with the DLT category gate. Runs against real Postgres +
// NATS; skips when infra is unreachable. Run with: make test-integration.
package sms

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/domains/compliance/ndnc"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
	"github.com/qeetgroup/qeet-notify/platform/telemetry"
)

const testEncKey = "test-enc-key"

type fakeProvider struct{ sends int }

func (f *fakeProvider) Send(_ context.Context, _ *Message) (*SendResult, error) {
	f.sends++
	return &SendResult{ProviderMessageID: "fake-sms-id"}, nil
}
func (f *fakeProvider) Name() string { return "fake" }

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func connect(t *testing.T) (*pgxpool.Pool, *messaging.Client) {
	t.Helper()
	dbURL := getenv("DATABASE_URL", "postgres://qeet-notify:qeet-notify@localhost:5433/qeet-notify?sslmode=disable")
	natsURL := getenv("NATS_URL", "nats://localhost:4222")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("integration: no database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("integration: database ping failed: %v", err)
	}
	nc, err := messaging.New(natsURL)
	if err != nil {
		pool.Close()
		t.Skipf("integration: no NATS: %v", err)
	}
	if err := nc.EnsureStreams(context.Background()); err != nil {
		pool.Close()
		nc.Close()
		t.Skipf("integration: ensure streams: %v", err)
	}
	return pool, nc
}

// seed creates a tenant, a subscriber with the given (encrypted) phone, an
// active 'sms' template, an approved DLT template of the given category whose
// body_regex matches the rendered body, and a queued notification.
func seed(t *testing.T, ctx context.Context, pool *pgxpool.Pool, phone, category string) (tenantID, subID, tmplID, notifID string) {
	t.Helper()
	uniq := time.Now().UnixNano()

	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, api_key_hash, api_key_prefix)
		 VALUES ('itest', $1, $2, 'itest') RETURNING id`,
		fmt.Sprintf("itest-%d", uniq), fmt.Sprintf("hash-%d", uniq),
	).Scan(&tenantID); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID) })

	// subscribers is RLS-forced (Module 36): seed inside a tenant-scoped tx.
	if err := database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, q database.Querier) error {
		return q.QueryRow(ctx,
			`INSERT INTO subscribers (tenant_id, external_id, phone_encrypted)
			 VALUES ($1, $2, pgp_sym_encrypt($3::text, $4)::text) RETURNING id`,
			tenantID, fmt.Sprintf("user-%d", uniq), phone, testEncKey,
		).Scan(&subID)
	}); err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO templates (tenant_id, name, channel, body, is_active)
		 VALUES ($1, $2, 'sms', 'Hello {{name}}', TRUE) RETURNING id`,
		tenantID, fmt.Sprintf("welcome-%d", uniq),
	).Scan(&tmplID); err != nil {
		t.Fatalf("seed template: %v", err)
	}

	// Approved DLT template whose regex matches the rendered body ("Hello Bob").
	if _, err := pool.Exec(ctx,
		`INSERT INTO dlt_templates
		    (tenant_id, channel, carrier, category, template_id_ext, template_name, body_regex, status, sender_id)
		 VALUES ($1, 'sms', 'all', $2, 'DLT-TEST-1', 'itest-dlt', '^Hello .*$', 'approved', 'QEET')`,
		tenantID, category,
	); err != nil {
		t.Fatalf("seed dlt template: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO notifications (tenant_id, subscriber_id, channel, template_id, status)
		 VALUES ($1, $2, 'sms', $3, 'queued') RETURNING id`,
		tenantID, subID, tmplID,
	).Scan(&notifID); err != nil {
		t.Fatalf("seed notification: %v", err)
	}
	return
}

func newWorker(pool *pgxpool.Pool, nc *messaging.Client, provider Provider) *Worker {
	return &Worker{pool: pool, js: nc.JS, primary: provider, encKey: testEncKey, log: telemetry.NewLogger("test")}
}

func statusOf(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(ctx, `SELECT status FROM notifications WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatalf("read status: %v", err)
	}
	return s
}

// TestPromotionalNDNCBlocked: a promotional SMS to an NDNC-registered number is
// blocked before the provider (and before the window check), status suppressed.
func TestPromotionalNDNCBlocked(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()
	ctx := context.Background()

	const phone = "+919000000001"
	tenantID, subID, tmplID, notifID := seed(t, ctx, pool, phone, "promotional")
	if err := ndnc.Register(ctx, pool, phone, "all", "manual"); err != nil {
		t.Fatalf("ndnc register: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM ndnc_registry`) })

	fake := &fakeProvider{}
	w := newWorker(pool, nc, fake)
	job := engine.ChannelJob{
		TenantID: tenantID, NotificationID: notifID, SubscriberID: subID,
		Channel: "sms", TemplateID: tmplID, Payload: map[string]any{"name": "Bob"},
	}
	var delay time.Duration
	if err := database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, _ database.Querier) error {
		d, e := w.handleJob(ctx, job)
		delay = d
		return e
	}); err != nil {
		t.Fatalf("handleJob: %v", err)
	}
	if delay != 0 {
		t.Errorf("NDNC-blocked should not defer; got delay=%v", delay)
	}
	if fake.sends != 0 {
		t.Errorf("NDNC-blocked: provider Send called %d times, want 0", fake.sends)
	}
	if s := statusOf(t, ctx, pool, notifID); s != "suppressed" {
		t.Errorf("NDNC-blocked: status = %q, want suppressed", s)
	}
}

// TestTransactionalExemptFromNDNC: transactional SMS bypasses the NDNC scrub and
// the promotional window, so it sends even to a DND-registered number.
func TestTransactionalExemptFromNDNC(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()
	ctx := context.Background()

	const phone = "+919000000002"
	tenantID, subID, tmplID, notifID := seed(t, ctx, pool, phone, "transactional")
	if err := ndnc.Register(ctx, pool, phone, "all", "manual"); err != nil {
		t.Fatalf("ndnc register: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM ndnc_registry`) })

	fake := &fakeProvider{}
	w := newWorker(pool, nc, fake)
	job := engine.ChannelJob{
		TenantID: tenantID, NotificationID: notifID, SubscriberID: subID,
		Channel: "sms", TemplateID: tmplID, Payload: map[string]any{"name": "Bob"},
	}
	var delay time.Duration
	if err := database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, _ database.Querier) error {
		d, e := w.handleJob(ctx, job)
		delay = d
		return e
	}); err != nil {
		t.Fatalf("handleJob: %v", err)
	}
	if delay != 0 {
		t.Errorf("transactional should not defer; got delay=%v", delay)
	}
	if fake.sends != 1 {
		t.Errorf("transactional: provider Send called %d times, want 1", fake.sends)
	}
	if s := statusOf(t, ctx, pool, notifID); s != "sent" {
		t.Errorf("transactional: status = %q, want sent", s)
	}
}
