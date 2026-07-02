package analytics

import (
	"context"
	"fmt"

	"github.com/qeetgroup/qeet-notify/platform/database"
)

// DeliveryTotals is the all-time aggregate funnel the console dashboard and
// analytics chart use. Event types that never fired will be 0.
type DeliveryTotals struct {
	Queued    int64 `json:"queued"`
	Sent      int64 `json:"sent"`
	Delivered int64 `json:"delivered"`
	Failed    int64 `json:"failed"`
	Opened    int64 `json:"opened"`
}

// QueryTotals returns all-time delivery event counts per funnel stage for a tenant.
func QueryTotals(ctx context.Context, q database.Querier, tenantID string) (DeliveryTotals, error) {
	rows, err := q.Query(ctx,
		`SELECT event_type, COUNT(*) FROM delivery_events
		 WHERE tenant_id = $1
		 GROUP BY event_type`,
		tenantID,
	)
	if err != nil {
		return DeliveryTotals{}, fmt.Errorf("query delivery totals: %w", err)
	}
	defer rows.Close()

	var t DeliveryTotals
	for rows.Next() {
		var evType string
		var cnt int64
		if err := rows.Scan(&evType, &cnt); err != nil {
			continue
		}
		switch evType {
		case "queued":
			t.Queued = cnt
		case "sent":
			t.Sent = cnt
		case "delivered":
			t.Delivered = cnt
		case "failed":
			t.Failed = cnt
		case "opened":
			t.Opened = cnt
		}
	}
	return t, rows.Err()
}
