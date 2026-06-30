package template

import (
	"context"
	"fmt"

	"github.com/aymerick/raymond"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Rendered holds the output of a template render for a single channel.
type Rendered struct {
	Subject string
	Body    string
}

// Fetch loads a template from the DB (no RLS needed — template_id is always
// fetched within the tenant's workflow run context).
func Fetch(ctx context.Context, pool *pgxpool.Pool, tenantID, templateID string) (subject, body string, err error) {
	err = pool.QueryRow(ctx,
		`SELECT COALESCE(subject,''), body FROM templates
		 WHERE id = $1 AND tenant_id = $2 AND is_active`,
		templateID, tenantID,
	).Scan(&subject, &body)
	if err != nil {
		return "", "", fmt.Errorf("fetch template %s: %w", templateID, err)
	}
	return subject, body, nil
}

// Render executes a Handlebars template string against the given data context.
func Render(tmpl string, data map[string]any) (string, error) {
	out, err := raymond.Render(tmpl, data)
	if err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return out, nil
}

// RenderEmail fetches a template from the DB and renders both subject and body.
func RenderEmail(ctx context.Context, pool *pgxpool.Pool, tenantID, templateID string, data map[string]any) (*Rendered, error) {
	subject, body, err := Fetch(ctx, pool, tenantID, templateID)
	if err != nil {
		return nil, err
	}

	renderedSubject, err := Render(subject, data)
	if err != nil {
		return nil, fmt.Errorf("render subject: %w", err)
	}
	renderedBody, err := Render(body, data)
	if err != nil {
		return nil, fmt.Errorf("render body: %w", err)
	}
	return &Rendered{Subject: renderedSubject, Body: renderedBody}, nil
}
