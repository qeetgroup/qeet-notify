//go:build integration

// Package integration exercises qeet-notify's workflow engine and scheduler
// against real Postgres + NATS JetStream. Run with: make test-integration
// (requires `make infra-up` + `make migrate-up`). Tests skip automatically when
// the infrastructure is unreachable.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/qeetgroup/qeet-notify/domains/scheduler"
	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
	"github.com/qeetgroup/qeet-notify/platform/telemetry"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// connect dials Postgres + NATS, skipping the test if either is unavailable.
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

// seed creates a tenant + subscriber + template and returns their IDs. The
// tenant is deleted (cascading) via t.Cleanup.
func seed(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (tenantID, subID, tmplID string) {
	t.Helper()
	uniq := time.Now().UnixNano()

	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, api_key_hash, api_key_prefix)
		 VALUES ('itest', $1, $2, 'itest') RETURNING id`,
		fmt.Sprintf("itest-%d", uniq), fmt.Sprintf("hash-%d", uniq),
	).Scan(&tenantID); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	// subscribers is RLS-forced (Module 36): seed inside a tenant-scoped tx.
	if err := database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, q database.Querier) error {
		return q.QueryRow(ctx,
			`INSERT INTO subscribers (tenant_id, external_id) VALUES ($1, 'user-1') RETURNING id`,
			tenantID,
		).Scan(&subID)
	}); err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO templates (tenant_id, name, channel, body)
		 VALUES ($1, 'welcome', 'email', 'Hello {{name}}') RETURNING id`,
		tenantID,
	).Scan(&tmplID); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	return tenantID, subID, tmplID
}

func createWorkflow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, trigger string, steps []engine.Step) {
	t.Helper()
	stepsJSON, _ := json.Marshal(steps)
	if _, err := pool.Exec(ctx,
		`INSERT INTO workflows (tenant_id, name, trigger_event, steps)
		 VALUES ($1, 'wf', $2, $3)`,
		tenantID, trigger, stepsJSON,
	); err != nil {
		t.Fatalf("create workflow: %v", err)
	}
}

func countNotifications(t *testing.T, ctx context.Context, pool *pgxpool.Pool, runID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE workflow_run_id = $1`, runID,
	).Scan(&n); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	return n
}

// TestWorkflowDelayResume covers the full delay→pause→scheduler→resume→complete
// lifecycle that was previously broken (delays never resumed).
func TestWorkflowDelayResume(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()

	ctx := context.Background()
	tenantID, subID, tmplID := seed(t, ctx, pool)

	createWorkflow(t, ctx, pool, tenantID, "itest.delay", []engine.Step{
		{ID: "s1", Type: engine.StepTypeChannel, Channel: "email", TemplateID: tmplID},
		{ID: "s2", Type: engine.StepTypeDelay, DelaySeconds: 3600},
		{ID: "s3", Type: engine.StepTypeChannel, Channel: "email", TemplateID: tmplID},
	})

	log := telemetry.NewLogger("test")
	eng := engine.New(pool, nc, log)

	// 1. New run dispatches the first channel then pauses on the delay.
	ev := engine.Event{TenantID: tenantID, SubscriberID: subID, Event: "itest.delay", Payload: map[string]any{}}
	if err := eng.ProcessEvent(ctx, ev); err != nil {
		t.Fatalf("ProcessEvent: %v", err)
	}

	var runID, status string
	var stepIdx int
	var resumeAt *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT id, status, current_step_index, resume_at FROM workflow_runs
		 WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1`, tenantID,
	).Scan(&runID, &status, &stepIdx, &resumeAt); err != nil {
		t.Fatalf("load run: %v", err)
	}
	if status != "running" {
		t.Errorf("after delay: status = %q, want running", status)
	}
	if stepIdx != 2 {
		t.Errorf("after delay: current_step_index = %d, want 2", stepIdx)
	}
	if resumeAt == nil {
		t.Error("after delay: resume_at is nil, want a future timestamp")
	}
	if got := countNotifications(t, ctx, pool, runID); got != 1 {
		t.Errorf("after delay: %d notifications, want 1 (first channel only)", got)
	}

	// 2. Scheduler picks up the due run and republishes a resume event.
	if _, err := pool.Exec(ctx,
		`UPDATE workflow_runs SET resume_at = NOW() - interval '1 second' WHERE id = $1`, runID,
	); err != nil {
		t.Fatalf("force due: %v", err)
	}

	consName := fmt.Sprintf("itest-events-%d", time.Now().UnixNano())
	cons, err := nc.JS.CreateOrUpdateConsumer(ctx, "NOTIFY_EVENTS", jetstream.ConsumerConfig{
		Name:           consName,
		AckPolicy:      jetstream.AckExplicitPolicy,
		FilterSubjects: []string{messaging.EventSubject(tenantID)},
	})
	if err != nil {
		t.Fatalf("create test consumer: %v", err)
	}
	defer func() { _ = nc.JS.DeleteConsumer(context.Background(), "NOTIFY_EVENTS", consName) }()

	scheduler.New(pool, nc, log).Tick(ctx)

	batch, err := cons.Fetch(1, jetstream.FetchMaxWait(5*time.Second))
	if err != nil {
		t.Fatalf("fetch resume event: %v", err)
	}
	var resumeEvt engine.Event
	got := 0
	for m := range batch.Messages() {
		_ = json.Unmarshal(m.Data(), &resumeEvt)
		_ = m.Ack()
		got++
	}
	if got != 1 {
		t.Fatalf("scheduler published %d resume events, want 1", got)
	}
	if resumeEvt.RunID != runID {
		t.Errorf("resume event RunID = %q, want %q", resumeEvt.RunID, runID)
	}

	if err := pool.QueryRow(ctx, `SELECT resume_at FROM workflow_runs WHERE id = $1`, runID).Scan(&resumeAt); err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if resumeAt != nil {
		t.Error("scheduler did not clear resume_at")
	}

	// 3. Resuming the run runs the remaining steps to completion.
	if err := eng.Resume(ctx, runID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT status, current_step_index FROM workflow_runs WHERE id = $1`, runID,
	).Scan(&status, &stepIdx); err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if status != "completed" {
		t.Errorf("after resume: status = %q, want completed", status)
	}
	if stepIdx != 3 {
		t.Errorf("after resume: current_step_index = %d, want 3", stepIdx)
	}
	if got := countNotifications(t, ctx, pool, runID); got != 2 {
		t.Errorf("after resume: %d notifications, want 2 (both channels)", got)
	}
}

// TestWorkflowCondition verifies a condition step branches by jumping to a step
// by ID, skipping the steps in between.
func TestWorkflowCondition(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()

	ctx := context.Background()
	tenantID, subID, tmplID := seed(t, ctx, pool)

	// gold tier → jump to the "gold" inapp step, skipping the default email step.
	createWorkflow(t, ctx, pool, tenantID, "itest.cond", []engine.Step{
		{ID: "cond", Type: engine.StepTypeCondition, Condition: `tier == "gold"`, TrueStep: "gold"},
		{ID: "default", Type: engine.StepTypeChannel, Channel: "email", TemplateID: tmplID},
		{ID: "gold", Type: engine.StepTypeChannel, Channel: "inapp", TemplateID: tmplID},
	})

	eng := engine.New(pool, nc, telemetry.NewLogger("test"))
	ev := engine.Event{TenantID: tenantID, SubscriberID: subID, Event: "itest.cond", Payload: map[string]any{"tier": "gold"}}
	if err := eng.ProcessEvent(ctx, ev); err != nil {
		t.Fatalf("ProcessEvent: %v", err)
	}

	var runID, status, channel string
	if err := pool.QueryRow(ctx,
		`SELECT id, status FROM workflow_runs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1`, tenantID,
	).Scan(&runID, &status); err != nil {
		t.Fatalf("load run: %v", err)
	}
	if status != "completed" {
		t.Errorf("status = %q, want completed", status)
	}

	rows, err := pool.Query(ctx, `SELECT channel FROM notifications WHERE workflow_run_id = $1`, runID)
	if err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	defer rows.Close()
	var channels []string
	for rows.Next() {
		_ = rows.Scan(&channel)
		channels = append(channels, channel)
	}
	if len(channels) != 1 || channels[0] != "inapp" {
		t.Errorf("gold branch dispatched %v, want exactly [inapp] (email step skipped)", channels)
	}
}
