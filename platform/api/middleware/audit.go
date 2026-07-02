package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// Audit records every mutating (non-GET) request under /v1 to the audit_logs
// table as a best-effort audit trail (Module 29). It runs after Auth +
// ScopeGuard, so tenant + actor context is available. The high-volume event
// intake (POST /v1/events) is excluded — the audit trail is for management/
// configuration changes, not per-notification traffic. Audit failures never
// block the request; the write uses a detached context so it still records even
// if the client disconnects.
//
// Field-level old/new value diffs are intentionally out of scope here (a later
// enhancement); this captures who did what to which resource, when, from where.
func Audit(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			pattern := chi.RouteContext(r.Context()).RoutePattern()
			if strings.HasSuffix(pattern, "/events") {
				return // exclude the hot-path event intake
			}
			tenantID, ok := TenantFromContext(r.Context())
			if !ok || tenantID == "" {
				return
			}
			actorType, actorID := ActorFromContext(r.Context())

			var resArg any
			if id := routeResourceID(r); isUUID(id) {
				resArg = id
			}

			// Detach from the request context so the audit row is still written
			// if the client has disconnected. The insert runs in its own
			// tenant-scoped tx (independent of the request tx) so it satisfies
			// RLS and is durable regardless of the request outcome.
			ctx := context.WithoutCancel(r.Context())
			_ = database.RunInTenant(ctx, pool, tenantID, func(ctx context.Context, q database.Querier) error {
				_, err := q.Exec(ctx,
					`INSERT INTO audit_logs
					    (tenant_id, actor_type, actor_id, action, resource_type, resource_id, new_value, ip_address, user_agent)
					 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
					tenantID, actorType, actorID,
					r.Method+" "+pattern, // action, e.g. "DELETE /v1/templates/{id}"
					resourceType(pattern),
					resArg,
					[]byte(fmt.Sprintf(`{"status":%d}`, ww.Status())), // new_value
					ClientIP(r), r.UserAgent(),
				)
				return err
			})
		})
	}
}

// resourceType returns the primary resource segment from a /v1 route pattern,
// e.g. "/v1/templates/{id}/publish" → "templates".
func resourceType(pattern string) string {
	p := strings.TrimPrefix(pattern, "/v1/")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		return "unknown"
	}
	return p
}

// routeResourceID returns the first URL path param that looks like a resource
// id (chi captures them by name: id, subscriberID, templateID, ...).
func routeResourceID(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	for _, v := range rctx.URLParams.Values {
		if isUUID(v) {
			return v
		}
	}
	return ""
}

// ClientIP returns the request's client IP (chi's RealIP middleware normalizes
// r.RemoteAddr upstream), stripping any port.
func ClientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}
