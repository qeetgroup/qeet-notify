//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

func rlsSeedTenant(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	uniq := time.Now().UnixNano()
	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, api_key_hash, api_key_prefix)
		 VALUES ('rls', $1, $2, 'rls') RETURNING id`,
		fmt.Sprintf("rls-%d", uniq), fmt.Sprintf("rlshash-%d", uniq),
	).Scan(&id); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id) })
	return id
}

func rlsSeedSubscriber(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, extID string) string {
	t.Helper()
	var subID string
	if err := database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, q database.Querier) error {
		return q.QueryRow(ctx,
			`INSERT INTO subscribers (tenant_id, external_id) VALUES ($1, $2) RETURNING id`,
			tenantID, extID,
		).Scan(&subID)
	}); err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}
	return subID
}

// countAsRole runs a subscribers count on one connection as the given
// non-superuser role, optionally scoped to a tenant via app.tenant_id.
// Superusers bypass RLS entirely, so enforcement can only be observed as a
// non-superuser role (which is how the app connects in production).
func countAsRole(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role, tenantID, whereID string) int {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if tenantID != "" {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, TRUE)`, tenantID); err != nil {
			t.Fatalf("set tenant: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+role); err != nil {
		t.Fatalf("set role: %v", err)
	}

	var n int
	if whereID != "" {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM subscribers WHERE id = $1`, whereID).Scan(&n); err != nil {
			t.Fatalf("count where: %v", err)
		}
	} else {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM subscribers`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
	}
	return n
}

// TestRLSCrossTenantIsolation proves FORCE ROW LEVEL SECURITY is real (Module 36 /
// PRD US-04): as a non-superuser role, a query that omits WHERE tenant_id
// returns only the current tenant's rows, and a query with no tenant set
// returns none.
func TestRLSCrossTenantIsolation(t *testing.T) {
	pool, nc := connect(t)
	defer pool.Close()
	defer nc.Close()
	ctx := context.Background()

	role := fmt.Sprintf("qeet_rls_probe_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `CREATE ROLE `+role+` NOSUPERUSER`); err != nil {
		t.Skipf("cannot create probe role (need CREATEROLE): %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `REVOKE ALL ON subscribers FROM `+role)
		_, _ = pool.Exec(context.Background(), `DROP ROLE IF EXISTS `+role)
	})
	if _, err := pool.Exec(ctx, `GRANT SELECT ON subscribers TO `+role); err != nil {
		t.Fatalf("grant: %v", err)
	}

	tenantA := rlsSeedTenant(t, ctx, pool)
	tenantB := rlsSeedTenant(t, ctx, pool)
	subA := rlsSeedSubscriber(t, ctx, pool, tenantA, "user-a")
	_ = rlsSeedSubscriber(t, ctx, pool, tenantB, "user-b")

	// Scoped to tenant B, an unfiltered count sees only B's own row.
	if got := countAsRole(t, ctx, pool, role, tenantB, ""); got != 1 {
		t.Errorf("RLS leak: unfiltered count as tenant B = %d, want 1 (only B's own row)", got)
	}
	// Tenant B cannot see tenant A's subscriber even addressing it by id.
	if got := countAsRole(t, ctx, pool, role, tenantB, subA); got != 0 {
		t.Errorf("RLS leak: tenant B sees tenant A's subscriber (%d rows), want 0", got)
	}
	// With no tenant set, RLS returns nothing.
	if got := countAsRole(t, ctx, pool, role, "", ""); got != 0 {
		t.Errorf("RLS not enforced: count with no tenant = %d, want 0", got)
	}
}
