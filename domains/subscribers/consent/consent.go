// Package consent implements the DPDP append-only consent ledger (Modules 22 &
// 34): every opt-in/opt-out is recorded with its source, purpose and policy
// version, and the full history is exposed for Data-Subject-Rights (access /
// export) requests.
package consent

import (
	"context"
	"fmt"
	"time"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// Entry is a single consent change to append to the ledger.
type Entry struct {
	TenantID     string
	SubscriberID string
	Channel      string
	Category     string
	OptedIn      bool
	Source       string // api | preference_center | import | unsubscribe_link | system
	Purpose      string
	Version      string
	Actor        string // api key fingerprint | "subscriber" | "system"
	IP           string
}

// Record appends a consent event to the ledger. The ledger is append-only —
// callers never update or delete rows.
func Record(ctx context.Context, q database.Querier, e Entry) error {
	if e.Category == "" {
		e.Category = "all"
	}
	if e.Source == "" {
		e.Source = "api"
	}
	_, err := q.Exec(ctx,
		`INSERT INTO consent_ledger
		    (tenant_id, subscriber_id, channel, category, opted_in, source, purpose, version, actor, ip_address)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		e.TenantID, e.SubscriberID, e.Channel, e.Category, e.OptedIn,
		e.Source, nilIfEmpty(e.Purpose), nilIfEmpty(e.Version), nilIfEmpty(e.Actor), nilIfEmpty(e.IP),
	)
	if err != nil {
		return fmt.Errorf("consent record: %w", err)
	}
	return nil
}

// Item is one historical consent entry returned by History (JSON-tagged for
// DSR export responses).
type Item struct {
	Channel   string    `json:"channel"`
	Category  string    `json:"category"`
	OptedIn   bool      `json:"opted_in"`
	Source    string    `json:"source"`
	Purpose   *string   `json:"purpose,omitempty"`
	Version   *string   `json:"version,omitempty"`
	Actor     *string   `json:"actor,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// History returns a subscriber's full consent trail, newest first.
func History(ctx context.Context, q database.Querier, tenantID, subscriberID string) ([]Item, error) {
	rows, err := q.Query(ctx,
		`SELECT channel, category, opted_in, source, purpose, version, actor, created_at
		 FROM consent_ledger
		 WHERE tenant_id = $1 AND subscriber_id = $2
		 ORDER BY created_at DESC`,
		tenantID, subscriberID,
	)
	if err != nil {
		return nil, fmt.Errorf("consent history: %w", err)
	}
	defer rows.Close()

	history := []Item{}
	for rows.Next() {
		var r Item
		if err := rows.Scan(&r.Channel, &r.Category, &r.OptedIn, &r.Source,
			&r.Purpose, &r.Version, &r.Actor, &r.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, r)
	}
	return history, rows.Err()
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
