package storage

import (
	"fmt"
	"time"
)

// PresignURL builds a time-limited URL for direct object access.
// In production this delegates to S3Store.PresignGet; this helper adds
// a human-readable expiry field to the returned metadata.
type PresignedURL struct {
	URL       string
	ExpiresAt time.Time
}

// String implements fmt.Stringer for logging.
func (p PresignedURL) String() string {
	return fmt.Sprintf("%s (expires %s)", p.URL, p.ExpiresAt.Format(time.RFC3339))
}
