package routing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Record holds a decrypted provider config row from provider_configs.
type Record struct {
	ProviderName string
	Config       []byte // decrypted JSON credentials
	Priority     int
}

// Load returns active provider configs for a tenant + channel, sorted by priority.
// Credentials are decrypted in-DB via notify_decrypt(). Returns an empty slice
// when no tenant-specific configs exist (caller should fall back to static providers).
func Load(ctx context.Context, pool *pgxpool.Pool, tenantID, channel, encKey string) ([]Record, error) {
	rows, err := pool.Query(ctx,
		`SELECT provider, notify_decrypt(config_encrypted, $3), priority
		 FROM provider_configs
		 WHERE tenant_id = $1 AND channel = $2 AND is_active
		 ORDER BY priority ASC`,
		tenantID, channel, encKey,
	)
	if err != nil {
		return nil, fmt.Errorf("routing.Load: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var configJSON string
		if err := rows.Scan(&r.ProviderName, &configJSON, &r.Priority); err != nil {
			return nil, fmt.Errorf("routing.Load scan: %w", err)
		}
		r.Config = []byte(configJSON)
		records = append(records, r)
	}
	return records, rows.Err()
}
