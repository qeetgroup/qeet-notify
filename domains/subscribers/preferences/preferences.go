package preferences

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// IsOptedIn returns false if the subscriber has opted out of the given channel+category.
// Falls back to checking channel='all' and category='all' if a specific row is missing.
func IsOptedIn(ctx context.Context, q database.Querier, tenantID, subscriberID, channel, category string) (bool, error) {
	var optedIn bool
	err := q.QueryRow(ctx,
		`SELECT is_opted_in FROM preferences
		 WHERE tenant_id = $1 AND subscriber_id = $2
		   AND channel IN ($3, 'all')
		   AND category IN ($4, 'all')
		 ORDER BY
		   CASE WHEN channel = $3 THEN 0 ELSE 1 END,
		   CASE WHEN category = $4 THEN 0 ELSE 1 END
		 LIMIT 1`,
		tenantID, subscriberID, channel, category,
	).Scan(&optedIn)
	if err != nil {
		// No preference row = opted in by default.
		return true, nil //nolint:nilerr
	}
	return optedIn, nil
}

// IsSuppressed checks whether the hashed value (email or phone) is on the suppression list.
func IsSuppressed(ctx context.Context, q database.Querier, tenantID, channel, plainValue string) (bool, error) {
	hash := hashValue(plainValue)
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM suppressions
			WHERE tenant_id = $1 AND channel = $2 AND value_hash = $3
		)`,
		tenantID, channel, hash,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("suppression check: %w", err)
	}
	return exists, nil
}

// AddSuppression inserts a suppression record (idempotent).
func AddSuppression(ctx context.Context, q database.Querier, tenantID, channel, plainValue, reason string) error {
	hash := hashValue(plainValue)
	_, err := q.Exec(ctx,
		`INSERT INTO suppressions (tenant_id, channel, value_hash, reason)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`,
		tenantID, channel, hash, reason,
	)
	return err
}

// Unsubscribe sets is_opted_in = false for all channels (or a specific channel).
func Unsubscribe(ctx context.Context, q database.Querier, tenantID, subscriberID, channel string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO preferences (tenant_id, subscriber_id, channel, category, is_opted_in)
		 VALUES ($1, $2, $3, 'all', FALSE)
		 ON CONFLICT (tenant_id, subscriber_id, channel, category)
		 DO UPDATE SET is_opted_in = FALSE, updated_at = NOW()`,
		tenantID, subscriberID, channel,
	)
	return err
}

// EraseSubscriber hard-deletes PII for DPDP right-to-erasure.
func EraseSubscriber(ctx context.Context, q database.Querier, tenantID, subscriberID string) error {
	_, err := q.Exec(ctx,
		`UPDATE subscribers
		 SET email_encrypted = NULL,
		     phone_encrypted = NULL,
		     whatsapp_id     = NULL,
		     push_tokens     = '[]',
		     metadata        = '{}',
		     is_deleted      = TRUE,
		     deleted_at      = NOW(),
		     updated_at      = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		subscriberID, tenantID,
	)
	return err
}

func hashValue(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
