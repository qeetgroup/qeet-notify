package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	platformnats "github.com/qeetgroup/qeet-notify/internal/platform/nats"
	"github.com/qeetgroup/qeet-notify/internal/preference"
)

// Engine consumes NOTIFY_EVENTS, resolves the matching workflow, and dispatches
// ChannelJob messages to the per-channel NATS streams.
type Engine struct {
	pool *pgxpool.Pool
	nc   *platformnats.Client
	log  zerolog.Logger
}

func New(pool *pgxpool.Pool, nc *platformnats.Client, log zerolog.Logger) *Engine {
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

	wf, err := e.lookupWorkflow(ctx, ev.TenantID, ev.Event)
	if err != nil {
		return err
	}
	if wf == nil {
		// No workflow matches this event — not an error, just ack and move on.
		return nil
	}

	runID, err := e.createRun(ctx, ev, wf)
	if err != nil {
		return err
	}

	return e.executeSteps(ctx, ev, wf.Steps, runID)
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

func (e *Engine) executeSteps(ctx context.Context, ev Event, steps []Step, runID string) error {
	for _, step := range steps {
		switch step.Type {
		case StepTypeChannel:
			if err := e.dispatchChannel(ctx, ev, step, runID); err != nil {
				return err
			}
		case StepTypeDelay:
			// Persist resume_at; a separate ticker re-enqueues the run.
			resumeAt := time.Now().Add(time.Duration(step.DelaySeconds) * time.Second)
			_, err := e.pool.Exec(ctx,
				`UPDATE workflow_runs SET resume_at = $1, updated_at = NOW() WHERE id = $2`,
				resumeAt, runID,
			)
			if err != nil {
				return fmt.Errorf("set resume_at: %w", err)
			}
			return nil // pause here; ticker picks up the remaining steps
		case StepTypeCondition:
			// Simple condition evaluation is a future enhancement; treat as pass-through.
		}
	}

	_, err := e.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = 'completed', updated_at = NOW() WHERE id = $1`,
		runID,
	)
	return err
}

func (e *Engine) dispatchChannel(ctx context.Context, ev Event, step Step, runID string) error {
	// Check subscriber preference before dispatching.
	category, _ := ev.Payload["category"].(string)
	if category == "" {
		category = "all"
	}
	optedIn, err := preference.IsOptedIn(ctx, e.pool, ev.TenantID, ev.SubscriberID, step.Channel, category)
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

	subject := platformnats.ChannelSubject(ev.TenantID, step.Channel)
	_, err = e.nc.JS.Publish(ctx, subject, data)
	return err
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
