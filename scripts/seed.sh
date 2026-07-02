#!/usr/bin/env bash
# seed.sh — populate the dev database with sample data.
# Requires: running Postgres (make infra-up) + migrations applied (make migrate-up).
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://qeet-notify:qeet-notify@localhost:5433/qeet-notify?sslmode=disable}"

echo "==> Seeding tenants..."
psql "$DATABASE_URL" <<'SQL'
INSERT INTO tenants (name, slug, api_key_hash, api_key_prefix)
VALUES
  ('Acme Corp',  'acme',  'dev-hash-acme',  'dev_acme'),
  ('Demo Tenant','demo',  'dev-hash-demo',  'dev_demo')
ON CONFLICT (slug) DO NOTHING;
SQL

echo "==> Seeding done."
