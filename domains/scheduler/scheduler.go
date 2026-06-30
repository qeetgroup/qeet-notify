package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/domains/workflows/engine"
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
func (s *Scheduler) Tick(ctx context.Context) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id FROM workflow_runs
		 WHERE status = 'running' AND resume_at IS NOT NULL AND resume_at <= NOW()
		 ORDER BY resume_at
		 LIMIT $1`,
		s.batch,
	)
	if err != nil {
		s.log.Error().Err(err).Msg("scheduler query due runs")
		return
	}

	type due struct{ runID, tenantID string }
	var dues []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.runID, &d.tenantID); err != nil {
			continue
		}
		dues = append(dues, d)
	}
	rows.Close()

	for _, d := range dues {
		// Clear resume_at before re-enqueueing so a slow engine never gets the
		// same run published twice.
		if _, err := s.pool.Exec(ctx,
			`UPDATE workflow_runs SET resume_at = NULL, updated_at = NOW() WHERE id = $1`,
			d.runID,
		); err != nil {
			s.log.Error().Err(err).Str("run", d.runID).Msg("scheduler clear resume_at")
			continue
		}
		payload, _ := json.Marshal(engine.Event{TenantID: d.tenantID, RunID: d.runID})
		if _, err := s.nc.JS.Publish(ctx, messaging.EventSubject(d.tenantID), payload); err != nil {
			s.log.Error().Err(err).Str("run", d.runID).Msg("scheduler publish resume")
		}
	}
}
