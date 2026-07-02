// Package ndnc implements NDNC/DND scrubbing (Module 32): the national
// Do-Not-Call registry that promotional SMS must be checked against before
// sending. Transactional traffic is exempt (the caller decides based on the
// matched DLT template category). Numbers are matched by SHA-256 hash.
package ndnc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// IsRegistered reports whether a phone number is on the national NDNC/DND
// registry. Any registered row for the number blocks promotional traffic.
func IsRegistered(ctx context.Context, q database.Querier, phone string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ndnc_registry WHERE phone_hash = $1)`,
		hashPhone(phone),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ndnc lookup: %w", err)
	}
	return exists, nil
}

// Register adds a phone number to the NDNC registry (idempotent). Used by the
// operator import / TRAI sync.
func Register(ctx context.Context, q database.Querier, phone, category, source string) error {
	if category == "" {
		category = "all"
	}
	if source == "" {
		source = "manual"
	}
	_, err := q.Exec(ctx,
		`INSERT INTO ndnc_registry (phone_hash, category, source)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		hashPhone(phone), category, source,
	)
	if err != nil {
		return fmt.Errorf("ndnc register: %w", err)
	}
	return nil
}

func hashPhone(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
