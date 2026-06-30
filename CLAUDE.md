# qeet-notify

Multi-channel transactional notification platform. Go 1.25 + chi v5 + PostgreSQL 17 + NATS JetStream 2.10 + Redis 7.

PRD: [../qeet-files/qeet-notify/Product_Requirement_Document.md](../qeet-files/qeet-notify/Product_Requirement_Document.md)  
TAD: [../qeet-files/qeet-notify/Technical_Architecture_Document.md](../qeet-files/qeet-notify/Technical_Architecture_Document.md)

## Quick commands

```bash
# Prerequisites (run once)
nvm use node   # Node >=25 for frontend
make infra-up  # Start Postgres, NATS, Redis, MinIO via Docker

# Copy and populate .env
cp .env.example .env

# Database
make migrate-up    # Apply all pending migrations
make db-up         # Alias for docker-compose up postgres
make db-reset      # Drop + recreate + migrate-up (dev only)

# Development
make dev           # Run qeet-notify-api with live reload

# Build
make build         # Build all binaries to bin/

# Test
make test                  # Unit tests
make test-integration      # Integration tests (needs running infra)

# Lint / format
make lint          # golangci-lint
make fmt           # gofmt + goimports

# Migrations
make migrate-up
make migrate-down n=1
make migrate-version
```

## Architecture

```
cmd/
  server/      → qeet-notify-api        (HTTP API)
  worker/      → qeet-notify-worker     (-channel=email|sms|whatsapp|push|webhook)
  workflow/    → qeet-notify-workflow   (DAG engine; consumes NATS NOTIFY_EVENTS)
  sse/         → qeet-notify-sse        (SSE long-poll; scales independently)
  analytics/   → qeet-notify-analytics  (aggregates delivery_events → TimescaleDB)
  migrate/     → qeet-notify-migrate    (golang-migrate CLI runner)

internal/
  api/          handler/ + middleware/
  workflow/     DAG executor + delay scheduler (Redis sorted set)
  channels/     email/ sms/ whatsapp/ push/ inapp/ webhook/
  subscriber/   CRUD + Qeet ID federation
  template/     Handlebars rendering + locale resolution
  preference/   Opt-out matrix; suppression enforcement
  analytics/    TimescaleDB hypertable helpers
  india/        TRAI DLT regex matching, NDNC Bloom filter, DPDP erasure
  platform/     db/ nats/ cache/ config/ logger/ metrics/

migrations/     SQL pairs: 0001_*.up.sql / 0001_*.down.sql (never edit applied)
frontend/       Next.js 16 dashboard (pnpm workspace)
sdk/go/         Go SDK
sdk/typescript/ @qeet-notify/node + @qeet-notify/react
cli/            `qn` cobra CLI
```

## Key conventions

- **Tenant isolation**: `tenant_id` injected via `X-Qeet-Api-Key` → SHA-256 → DB lookup; set as `current_setting('app.tenant_id')` for RLS.
- **NATS subjects**: `qeet-notify.{tenant_id}.events`, `qeet-notify.{tenant_id}.channel.{channel}`
- **Migrations**: sequential integers, immutable once applied. Run `make migrate-up` after pull.
- **PII encryption**: `pgp_sym_encrypt(value, current_setting('app.enc_key'))` for email/phone in DB.
- **India DLT**: every outbound SMS must match a stored `dlt_templates.regex`; promotional window 10:00–21:00 IST.

## Infrastructure ports (local dev)

| Service | Port |
|---|---|
| qeet-notify-api | 8080 |
| PostgreSQL 17 | 5433 |
| NATS | 4222 / 8222 (monitor) |
| Redis 7 | 6379 |
| MinIO | 9000 / 9001 (console) |

## Deployment

See [deploy/](deploy/) — mirrors qeet-id deploy pattern: GHCR + SSH + `docker compose up -d`.
