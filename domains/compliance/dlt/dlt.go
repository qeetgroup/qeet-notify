package dlt

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ISTLocation is the Asia/Kolkata timezone.
var ISTLocation = mustLoadLocation("Asia/Kolkata")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Fallback: UTC+5:30 if timezone DB is unavailable in container.
		return time.FixedZone("IST", 5*60*60+30*60)
	}
	return loc
}

// IsPromotionalWindowOpen returns true if the current IST time is within
// the TRAI-mandated promotional SMS window (10:00–21:00 IST).
func IsPromotionalWindowOpen() bool {
	now := time.Now().In(ISTLocation)
	h := now.Hour()
	return h >= 10 && h < 21
}

// ResumeAtNextWindow returns the next 10:00 IST if we're currently outside the window.
func ResumeAtNextWindow() time.Time {
	now := time.Now().In(ISTLocation)
	next := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, ISTLocation)
	if now.After(next) || now.Equal(next) {
		// 10:00 today already passed (or it's past 21:00), so schedule for tomorrow.
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// DLTTemplate is a loaded + compiled TRAI template.
type DLTTemplate struct {
	ID      string
	Carrier string
	Regex   *regexp.Regexp
}

// LoadApprovedTemplates fetches all approved DLT templates for a tenant + carrier.
func LoadApprovedTemplates(ctx context.Context, pool *pgxpool.Pool, tenantID, carrier string) ([]DLTTemplate, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, carrier, body_regex FROM dlt_templates
		 WHERE tenant_id = $1 AND channel = 'sms'
		   AND (carrier = $2 OR carrier = 'all')
		   AND status = 'approved'`,
		tenantID, carrier,
	)
	if err != nil {
		return nil, fmt.Errorf("load DLT templates: %w", err)
	}
	defer rows.Close()

	var templates []DLTTemplate
	for rows.Next() {
		var t DLTTemplate
		var rawRegex string
		if err := rows.Scan(&t.ID, &t.Carrier, &rawRegex); err != nil {
			return nil, err
		}
		compiled, err := regexp.Compile(rawRegex)
		if err != nil {
			continue // skip malformed regex; operator needs to fix in wizard
		}
		t.Regex = compiled
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// MatchTemplate returns the DLT template ID whose regex matches body, or "" if none match.
func MatchTemplate(templates []DLTTemplate, body string) string {
	for _, t := range templates {
		if t.Regex.MatchString(body) {
			return t.ID
		}
	}
	return ""
}
