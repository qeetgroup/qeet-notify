package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
)

type dashboardClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Iss   string `json:"iss"`
	Exp   int64  `json:"exp"`
}

type contextKeyDashboard struct{}

// DashboardUserFromContext extracts the authenticated dashboard user subject.
func DashboardUserFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(contextKeyDashboard{}).(string)
	return v, ok
}

// jwksCache caches the JWKS fetched from Qeet ID.
type jwksCache struct {
	mu      sync.RWMutex
	keySet  *jose.JSONWebKeySet
	fetched time.Time
}

var globalJWKS = &jwksCache{}

func (c *jwksCache) get(ctx context.Context, issuer string) (*jose.JSONWebKeySet, error) {
	c.mu.RLock()
	if c.keySet != nil && time.Since(c.fetched) < 5*time.Minute {
		ks := c.keySet
		c.mu.RUnlock()
		return ks, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	url := strings.TrimRight(issuer, "/") + "/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var ks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&ks); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}
	c.keySet = &ks
	c.fetched = time.Now()
	return &ks, nil
}

// DashboardAuth validates a Qeet ID JWT from the Authorization header.
// Used on dashboard-only routes (not the API-key routes).
func DashboardAuth(issuer string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				http.Error(w, `{"error":"missing Authorization"}`, http.StatusUnauthorized)
				return
			}

			ks, err := globalJWKS.get(r.Context(), issuer)
			if err != nil {
				http.Error(w, `{"error":"jwks unavailable"}`, http.StatusServiceUnavailable)
				return
			}

			tok, err := josejwt.ParseSigned(raw)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			var claims dashboardClaims
			for _, key := range ks.Keys {
				if err := tok.Claims(key, &claims); err == nil {
					break
				}
			}
			if claims.Sub == "" || claims.Iss != issuer || claims.Exp < time.Now().Unix() {
				http.Error(w, `{"error":"token invalid or expired"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyDashboard{}, claims.Sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
