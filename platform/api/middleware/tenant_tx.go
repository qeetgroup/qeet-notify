package middleware

import (
	"context"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// TenantTx runs each authenticated request inside a transaction that has
// app.tenant_id set (SET LOCAL), and exposes that transaction as the
// request-scoped Querier (database.FromContext). This is what makes PostgreSQL
// row-level security effective for the request path (Module 36): every handler
// query executes on a connection scoped to the caller's tenant. The tx commits
// on a <400 response and rolls back otherwise. Must run after Auth (which sets
// the tenant in context).
func TenantTx(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := TenantFromContext(r.Context())
			if !ok || tenantID == "" {
				next.ServeHTTP(w, r)
				return
			}

			tx, err := pool.Begin(r.Context())
			if err != nil {
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}
			committed := false
			defer func() {
				if !committed {
					_ = tx.Rollback(context.Background())
				}
			}()

			if _, err := tx.Exec(r.Context(), `SELECT set_config('app.tenant_id', $1, TRUE)`, tenantID); err != nil {
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}

			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			ctx := database.ContextWithQuerier(r.Context(), tx)
			next.ServeHTTP(ww, r.WithContext(ctx))

			if ww.Status() < http.StatusBadRequest {
				if err := tx.Commit(r.Context()); err == nil {
					committed = true
				}
			}
		})
	}
}
