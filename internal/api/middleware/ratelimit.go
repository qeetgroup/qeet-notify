package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit enforces a sliding-window request limit per tenant using Redis.
// limit = max requests per window duration.
func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := TenantFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			key := fmt.Sprintf("rl:%s", tenantID)
			ctx := r.Context()

			pipe := rdb.Pipeline()
			incr := pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, window)
			if _, err := pipe.Exec(ctx); err != nil {
				// fail open — don't block the request on Redis errors
				next.ServeHTTP(w, r)
				return
			}

			if incr.Val() > int64(limit) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
