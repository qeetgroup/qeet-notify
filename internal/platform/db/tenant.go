package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LookupTenantByAPIKeyHash returns the tenant UUID for a given SHA-256 API key hash.
// Returns ("", false, nil) when no matching tenant is found.
func LookupTenantByAPIKeyHash(ctx context.Context, pool *pgxpool.Pool, hash string) (string, bool, error) {
	var id string
	err := pool.QueryRow(ctx,
		`SELECT id FROM tenants WHERE api_key_hash = $1 LIMIT 1`,
		hash,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("tenant lookup: %w", err)
	}
	return id, true, nil
}

// WithTenant runs fn inside a transaction with RLS context set to tenantID.
// All DB queries inside fn benefit from row-level security automatically.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.tenant_id', $1, TRUE)`, tenantID,
	); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
