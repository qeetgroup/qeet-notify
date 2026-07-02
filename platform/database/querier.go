package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so data-access code can
// run against either the shared pool or a tenant-scoped transaction. Domain
// functions and handlers take a Querier (via FromContext) so that, when a
// request/job runs inside a tenant transaction, all queries execute on the
// connection that has app.tenant_id set — which is what makes RLS effective
// (Module 36).
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ctxKey int

const querierKey ctxKey = iota

// ContextWithQuerier stores the tenant-scoped querier for the current
// request/job so downstream code can retrieve it via FromContext.
func ContextWithQuerier(ctx context.Context, q Querier) context.Context {
	return context.WithValue(ctx, querierKey, q)
}

// FromContext returns the tenant-scoped querier bound to ctx, or fallback (the
// shared pool) when none is set (e.g. unauthenticated/bootstrap paths).
func FromContext(ctx context.Context, fallback Querier) Querier {
	if q, ok := ctx.Value(querierKey).(Querier); ok && q != nil {
		return q
	}
	return fallback
}

// RunInTenant runs fn inside a transaction with app.tenant_id set (SET LOCAL,
// so it clears automatically on commit/rollback). The tx is exposed both as the
// q argument and via the ctx passed to fn (ContextWithQuerier), so nested
// domain calls that use FromContext pick it up. Used by background jobs and
// one-off tenant-scoped operations outside the HTTP request path.
func RunInTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(ctx context.Context, q Querier) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tenant tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, TRUE)`, tenantID); err != nil {
		return fmt.Errorf("set app.tenant_id: %w", err)
	}
	if err := fn(ContextWithQuerier(ctx, tx), tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
