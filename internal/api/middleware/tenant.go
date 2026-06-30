package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

type contextKey string

const tenantIDKey contextKey = "tenantID"

// TenantFromContext extracts the tenant ID injected by Auth middleware.
func TenantFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(tenantIDKey).(string)
	return v, ok
}

// hashAPIKey returns hex(SHA-256(key)) for comparison against stored hashes.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// placeholder — real lookup wired in Step 2 when DB is ready
type TenantLookup func(ctx context.Context, keyHash string) (tenantID string, found bool, err error)

func Auth(lookup TenantLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Qeet-Api-Key")
			if key == "" {
				http.Error(w, `{"error":"missing X-Qeet-Api-Key"}`, http.StatusUnauthorized)
				return
			}
			tenantID, found, err := lookup(r.Context(), hashAPIKey(key))
			if err != nil || !found {
				http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
