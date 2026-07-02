package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
	"github.com/qeetgroup/qeet-notify/platform/database"
	"github.com/qeetgroup/qeet-notify/platform/messaging"
)

// Scheduler periodically re-enqueues workflow runs whose delay (resume_at) has
// elapsed so the workflow engine continues them from current_step_index. It is
// the canonical home for delay-resume; the engine pauses a run on a delay step
// and this binary wakes it back up.
type Scheduler struct {
	pool     *pgxpool.Pool
	nc       *messaging.Client
	log      zerolog.Logger
	interval time.Duration
	batch    int
}

func New(pool *pgxpool.Pool, nc *messaging.Client, log zerolog.Logger) *Scheduler {
	return &Scheduler{pool: pool, nc: nc, log: log, interval: 30 * time.Second, batch: 50}
}

// Run ticks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.log.Info().Dur("interval", s.interval).Msg("scheduler started")
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

// Tick performs one scan: re-enqueues every workflow run whose resume_at is due.
// Exported so it can be driven deterministically in tests.
//
// workflow_runs is RLS-forced (Module 36), so the scan is done per-tenant: we
// enumerate tenants (the tenants table is not RLS-scoped) and, inside a
// tenant-scoped tx, find and re-enqueue that tenant's due runs.
func (s *Scheduler) Tick(ctx context.Context) {
	tenantRows, err := s.pool.Query(ctx, `SELECT id FROM tenants`)
	if err != nil {
		s.log.Error().Err(err).Msg("scheduler list tenants")
		return
	}
	var tenantIDs []string
	for tenantRows.Next() {
		var id string
		if err := tenantRows.Scan(&id); err == nil {
			tenantIDs = append(tenantIDs, id)
		}
	}
	tenantRows.Close()

	for _, tid := range tenantIDs {
		err := database.RunInTenant(ctx, s.pool, tid, func(ctx context.Context, q database.Querier) error {
			// Keep the explicit tenant filter as well as the tenant-scoped tx:
			// RLS is only enforced when the app connects as a non-superuser role,
			// so the WHERE clause guarantees correct scoping either way.
			rows, err := q.Query(ctx,
				`SELECT id FROM workflow_runs
				 WHERE tenant_id = $1 AND status = 'running' AND resume_at IS NOT NULL AND resume_at <= NOW()
				 ORDER BY resume_at
				 LIMIT $2`,
				tid, s.batch,
			)
			if err != nil {
				return err
			}
			var runIDs []string
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					runIDs = append(runIDs, id)
				}
			}
			rows.Close()

			for _, runID := range runIDs {
				// Clear resume_at before re-enqueueing so a slow engine never
				// gets the same run published twice.
				if _, err := q.Exec(ctx,
					`UPDATE workflow_runs SET resume_at = NULL, updated_at = NOW() WHERE id = $1`,
					runID,
				); err != nil {
					s.log.Error().Err(err).Str("run", runID).Msg("scheduler clear resume_at")
					continue
				}
				payload, _ := json.Marshal(engine.Event{TenantID: tid, RunID: runID})
				if _, err := s.nc.JS.Publish(ctx, messaging.EventSubject(tid), payload); err != nil {
					s.log.Error().Err(err).Str("run", runID).Msg("scheduler publish resume")
				}
			}
			return nil
		})
		if err != nil {
			s.log.Error().Err(err).Str("tenant", tid).Msg("scheduler tenant tick")
		}
	}
}
