package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

type contextKey string

const (
	tenantIDKey   contextKey = "tenantID"
	scopeKey      contextKey = "apiKeyScope"
	apiKeyHashKey contextKey = "apiKeyHash"
)

// TenantFromContext extracts the tenant ID injected by Auth middleware.
func TenantFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(tenantIDKey).(string)
	return v, ok
}

// ScopeFromContext extracts the API key scope injected by Auth middleware.
// Returns "full" as the default when not set.
func ScopeFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(scopeKey).(string)
	if !ok || v == "" {
		return "full", false
	}
	return v, true
}

// ActorFromContext identifies who is making the request, for audit logging.
// Returns ("api_key", "apikey:<fingerprint>") when authenticated via an API key
// (the fingerprint is a short, stable prefix of the key hash — enough to
// attribute the action without storing the full credential hash), else
// ("system", "system").
func ActorFromContext(ctx context.Context) (actorType, actorID string) {
	if h, ok := ctx.Value(apiKeyHashKey).(string); ok && h != "" {
		if len(h) > 16 {
			h = h[:16]
		}
		return "api_key", "apikey:" + h
	}
	return "system", "system"
}

// hashAPIKey returns hex(SHA-256(key)) for comparison against stored hashes.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// TenantLookup resolves a key hash to tenant ID and scope.
// scope is one of: "full" | "read" | "send".
type TenantLookup func(ctx context.Context, keyHash string) (tenantID, scope string, found bool, err error)

func Auth(lookup TenantLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Qeet-Api-Key")
			if key == "" {
				http.Error(w, `{"error":"missing X-Qeet-Api-Key"}`, http.StatusUnauthorized)
				return
			}
			keyHash := hashAPIKey(key)
			tenantID, scope, found, err := lookup(r.Context(), keyHash)
			if err != nil || !found {
				http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
			ctx = context.WithValue(ctx, scopeKey, scope)
			ctx = context.WithValue(ctx, apiKeyHashKey, keyHash)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ScopeGuard enforces API key scope rules on every request:
//   - "full"  → unrestricted
//   - "read"  → GET requests only
//   - "send"  → POST /v1/events only
func ScopeGuard() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope, _ := ScopeFromContext(r.Context())
			switch scope {
			case "full":
				// unrestricted
			case "read":
				if r.Method != http.MethodGet {
					http.Error(w, `{"error":"scope 'read' only allows GET requests"}`, http.StatusForbidden)
					return
				}
			case "send":
				if !(r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/events")) {
					http.Error(w, `{"error":"scope 'send' only allows POST /v1/events"}`, http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
