package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/qeetgroup/qeet-notify/platform/messaging"
	"github.com/qeetgroup/qeet-notify/domains/subscribers/preferences"
)

// Engine consumes NOTIFY_EVENTS, resolves the matching workflow, and dispatches
// ChannelJob messages to the per-channel NATS streams.
type Engine struct {
	pool *pgxpool.Pool
	nc   *messaging.Client
	log  zerolog.Logger
}

func New(pool *pgxpool.Pool, nc *messaging.Client, log zerolog.Logger) *Engine {
	return &Engine{pool: pool, nc: nc, log: log}
}

// Run blocks and processes NOTIFY_EVENTS until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) error {
	cons, err := e.nc.JS.CreateOrUpdateConsumer(ctx, "NOTIFY_EVENTS", jetstream.ConsumerConfig{
		Name:          "workflow-engine",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	msgs, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer msgs.Stop()

	e.log.Info().Msg("workflow engine started")
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := msgs.Next()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			e.log.Error().Err(err).Msg("receive event")
			continue
		}

		if err := e.handle(ctx, msg); err != nil {
			e.log.Error().Err(err).Msg("handle event")
			msg.Nak() //nolint:errcheck
		} else {
			msg.Ack() //nolint:errcheck
		}
	}
}

func (e *Engine) handle(ctx context.Context, msg jetstream.Msg) error {
	var ev Event
	if err := json.Unmarshal(msg.Data(), &ev); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	// Resume signal from the scheduler: an existing run whose delay has elapsed.
	if ev.RunID != "" {
		return e.Resume(ctx, ev.RunID)
	}
	return e.ProcessEvent(ctx, ev)
}

// ProcessEvent runs a fresh workflow triggered by ev. It is a no-op (nil) when no
// active workflow matches the event.
func (e *Engine) ProcessEvent(ctx context.Context, ev Event) error {
	wf, err := e.lookupWorkflow(ctx, ev.TenantID, ev.Event)
	if err != nil {
		return err
	}
	if wf == nil {
		return nil
	}

	runID, err := e.createRun(ctx, ev, wf)
	if err != nil {
		return err
	}

	return e.executeFrom(ctx, ev, wf.Steps, runID, 0)
}

// Resume reloads a paused workflow run and continues it from current_step_index.
// Invoked when the scheduler re-enqueues a run whose delay (resume_at) has passed.
func (e *Engine) Resume(ctx context.Context, runID string) error {
	var tenantID, triggerEvent, payloadJSON, stepsJSON string
	var subscriberID string
	var startIdx int
	err := e.pool.QueryRow(ctx,
		`SELECT r.tenant_id, COALESCE(r.subscriber_id::text, ''), r.trigger_event,
		        r.trigger_payload::text, r.current_step_index, w.steps::text
		 FROM workflow_runs r
		 JOIN workflows w ON w.id = r.workflow_id
		 WHERE r.id = $1 AND r.status = 'running'`,
		runID,
	).Scan(&tenantID, &subscriberID, &triggerEvent, &payloadJSON, &startIdx, &stepsJSON)
	if err != nil {
		// Run was cancelled/completed concurrently, or no longer exists — nothing to do.
		return nil //nolint:nilerr
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return fmt.Errorf("parse resume payload: %w", err)
	}
	var steps []Step
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		return fmt.Errorf("parse resume steps: %w", err)
	}

	ev := Event{
		TenantID:     tenantID,
		SubscriberID: subscriberID,
		Event:        triggerEvent,
		Payload:      payload,
		RunID:        runID,
	}
	return e.executeFrom(ctx, ev, steps, runID, startIdx)
}

type dbWorkflow struct {
	ID    string
	Steps []Step
}

func (e *Engine) lookupWorkflow(ctx context.Context, tenantID, triggerEvent string) (*dbWorkflow, error) {
	var id, stepsJSON string
	err := e.pool.QueryRow(ctx,
		`SELECT id, steps FROM workflows
		 WHERE tenant_id = $1 AND trigger_event = $2 AND is_active
		 LIMIT 1`,
		tenantID, triggerEvent,
	).Scan(&id, &stepsJSON)
	if err != nil {
		// pgx.ErrNoRows → no workflow, return nil
		return nil, nil //nolint:nilerr
	}

	var steps []Step
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		return nil, fmt.Errorf("parse steps: %w", err)
	}
	return &dbWorkflow{ID: id, Steps: steps}, nil
}

func (e *Engine) createRun(ctx context.Context, ev Event, wf *dbWorkflow) (string, error) {
	payloadJSON, _ := json.Marshal(ev.Payload)
	var runID string
	err := e.pool.QueryRow(ctx,
		`INSERT INTO workflow_runs
		    (tenant_id, workflow_id, subscriber_id, trigger_event, trigger_payload)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		ev.TenantID, wf.ID, nilIfEmpty(ev.SubscriberID), ev.Event, payloadJSON,
	).Scan(&runID)
	return runID, err
}

// executeFrom drives a run's steps starting at index `start`. It walks an index
// pointer (not a plain range) so condition steps can branch to a step by ID. A
// delay step persists the next index + resume_at and returns, pausing the run
// until the scheduler re-enqueues it.
func (e *Engine) executeFrom(ctx context.Context, ev Event, steps []Step, runID string, start int) error {
	if start < 0 {
		start = 0
	}
	idxByID := make(map[string]int, len(steps))
	for i, s := range steps {
		if s.ID != "" {
			idxByID[s.ID] = i
		}
	}

	// Guard against cycles introduced via next_step / true_step / false_step.
	maxIterations := len(steps)*2 + 10

	i := start
	for visited := 0; i < len(steps); visited++ {
		if visited > maxIterations {
			return e.failRun(ctx, runID, "workflow exceeded max steps (cycle?)")
		}
		step := steps[i]
		switch step.Type {
		case StepTypeChannel:
			if err := e.dispatchChannel(ctx, ev, step, runID); err != nil {
				return err
			}
			i = nextIndex(idxByID, step, i)
		case StepTypeDelay:
			resumeAt := time.Now().Add(time.Duration(step.DelaySeconds) * time.Second)
			next := nextIndex(idxByID, step, i)
			_, err := e.pool.Exec(ctx,
				`UPDATE workflow_runs SET resume_at = $1, current_step_index = $2, updated_at = NOW()
				 WHERE id = $3`,
				resumeAt, next, runID,
			)
			if err != nil {
				return fmt.Errorf("set resume_at: %w", err)
			}
			return nil // pause; the scheduler re-enqueues this run from `next`
		case StepTypeCondition:
			result, err := EvalCondition(step.Condition, ev.Payload)
			if err != nil {
				e.log.Warn().Err(err).Str("condition", step.Condition).Msg("condition eval failed; treating as false")
				result = false
			}
			target := step.FalseStep
			if result {
				target = step.TrueStep
			}
			if target == "" {
				i++ // no branch target — fall through to the next step
				continue
			}
			ti, ok := idxByID[target]
			if !ok {
				i = len(steps) // unknown target id — end the run
				continue
			}
			i = ti
		default:
			i++ // unknown step type — skip
		}
	}

	_, err := e.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = 'completed', current_step_index = $1, updated_at = NOW()
		 WHERE id = $2`,
		len(steps), runID,
	)
	return err
}

// nextIndex returns the index of the step to run after `i`: the step named by
// step.NextStep if set and known, otherwise the next step in order.
func nextIndex(idxByID map[string]int, step Step, i int) int {
	if step.NextStep != "" {
		if ni, ok := idxByID[step.NextStep]; ok {
			return ni
		}
	}
	return i + 1
}

func (e *Engine) failRun(ctx context.Context, runID, reason string) error {
	_, err := e.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = 'failed', error = $1, updated_at = NOW() WHERE id = $2`,
		reason, runID,
	)
	return err
}

func (e *Engine) dispatchChannel(ctx context.Context, ev Event, step Step, runID string) error {
	// Check subscriber preference before dispatching.
	category, _ := ev.Payload["category"].(string)
	if category == "" {
		category = "all"
	}
	optedIn, err := preferences.IsOptedIn(ctx, e.pool, ev.TenantID, ev.SubscriberID, step.Channel, category)
	if err != nil {
		e.log.Warn().Err(err).Msg("preference check failed; defaulting to opted-in")
	}
	if !optedIn {
		e.log.Debug().Str("subscriber", ev.SubscriberID).Str("channel", step.Channel).Msg("preference skip")
		return nil
	}

	// Persist the notification record so workers can update its status.
	var notificationID string
	if err := e.pool.QueryRow(ctx,
		`INSERT INTO notifications
		    (tenant_id, workflow_run_id, subscriber_id, channel, template_id, status)
		 VALUES ($1, $2, $3, $4, $5, 'queued')
		 RETURNING id`,
		ev.TenantID, runID, nilIfEmpty(ev.SubscriberID), step.Channel, nilIfEmpty(step.TemplateID),
	).Scan(&notificationID); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	job := ChannelJob{
		TenantID:       ev.TenantID,
		NotificationID: notificationID,
		SubscriberID:   ev.SubscriberID,
		Channel:        step.Channel,
		TemplateID:     step.TemplateID,
		Payload:        ev.Payload,
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal channel job: %w", err)
	}

	subject := messaging.ChannelSubject(ev.TenantID, step.Channel)
	_, err = e.nc.JS.Publish(ctx, subject, data)
	return err
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
