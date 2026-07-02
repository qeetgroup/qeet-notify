# CLAUDE.md

`qeet-notify` is the **Qeet Notify** multi-channel transactional notification platform.

PRD: [../qeet-files/qeet-notify/Product_Requirement_Document.md](../qeet-files/qeet-notify/Product_Requirement_Document.md)  
TAD: [../qeet-files/qeet-notify/Technical_Architecture_Document.md](../qeet-files/qeet-notify/Technical_Architecture_Document.md)

## Quick commands

```bash
# Prerequisites (run once)
nvm use node   # Node >=25 for apps/console
make infra-up  # Start Postgres, NATS, Redis via Docker

# Copy and populate .env
cp .env.example .env

# Database
make migrate-up    # Apply all pending migrations
make db-reset      # Drop + recreate + migrate-up (dev only)

# Development
make dev           # Run qeet-notify-server with live reload
make dev-console   # Start TanStack Start console on :3010

# Build
make build         # Build all Go binaries to bin/

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
  server/      → qeet-notify-server    (HTTP API :8080 + SSE :8082 in one binary)
  worker/      → qeet-notify-worker    (-channel=email|sms|whatsapp|inapp|webhook)
  scheduler/   → qeet-notify-scheduler (delay/retry ticker — stub, coming soon)
  workflow/    → qeet-notify-workflow  (DAG engine; consumes NATS NOTIFY_EVENTS)
  analytics/   → qeet-notify-analytics (aggregates delivery_events → TimescaleDB)
  migrate/     → qeet-notify-migrate   (golang-migrate CLI runner)

domains/                   Business logic — bounded contexts
  analytics/               Delivery aggregation, Prometheus, TimescaleDB queries
  compliance/dlt/          TRAI DLT regex matching, promotional window, NDNC
  channels/                Channel queue workers (one per channel)
    email/                 NOTIFY_EMAIL worker + inline provider types
    sms/                   NOTIFY_SMS worker + DLT/NDNC gates
    whatsapp/              NOTIFY_WHATSAPP worker (Meta Cloud API)
    inapp/                 NOTIFY_INAPP worker (Redis pub/sub fan-out)
    webhook/               Outbound HMAC-signed webhook worker
    push/                  (stub — FCM/APNs not yet implemented)
  providers/               Pure vendor adapters (interface + implementations)
    email/                 SESProvider + ResendProvider + BuildProviders registry
    sms/                   MSG91Provider + TwoFactorProvider + BuildProviders
    whatsapp/              MetaProvider + BuildProviders
    push/                  (stub — fcm/.gitkeep + apns/.gitkeep)
  subscribers/
    federation/            Qeet ID user-event → subscriber sync
    preferences/           Opt-in/out matrix + DPDP erasure
  templates/rendering/     Handlebars template fetch + render
  workflows/engine/        DAG executor + delay step support

platform/                  Shared infrastructure (no business logic)
  api/handler/             HTTP route handlers
  api/middleware/          Auth (API key → tenant), rate-limit, OIDC dashboard auth
  cache/                   Redis client
  config/                  envconfig loader
  database/                pgxpool + tenant RLS helper (WithTenant)
  messaging/               NATS JetStream client + stream definitions
  observability/           zerolog logger
  events/                  (stub — shared event type contracts)
  security/                (stub — HMAC/crypto helpers)
  storage/                 (stub — MinIO client)

apps/console/              TanStack Start dashboard (:3010) — file-based routing, @qeetrix/ui

sdk/go/                    Public Go SDK
sdk/node/                  @qeet-notify/node TypeScript SDK
sdk/python/                (stub)
sdk/java/                  (stub)

packages/                  JS monorepo packages (stubs)
  ui/                      @qeet-notify/ui
  design-system/           @qeet-notify/design-system
  api-client/              @qeet-notify/api-client
  shared-types/            @qeet-notify/shared-types
  notification-sdk/        @qeet-notify/notification-sdk

api/openapi/               OpenAPI 3.1 spec (v1.yaml)
api/postman/               Postman collection (stub)
api/contracts/             API contract tests (stub)

migrations/                SQL pairs: 0001_*.up.sql / 0001_*.down.sql (never edit applied)
tests/integration/         Integration test suites
tests/e2e/                 End-to-end tests
tests/performance/         Load / performance tests
tests/architecture/        Architectural fitness functions
tools/                     Codegen, linting helpers
docs/                      Internal documentation
```

## Key conventions

- **Tenant isolation**: `tenant_id` injected via `X-Qeet-Api-Key` → SHA-256 → DB lookup; set as `current_setting('app.tenant_id')` for RLS.
- **NATS subjects**: `qeet-notify.{tenant_id}.events`, `qeet-notify.{tenant_id}.channel.{channel}`
- **Migrations**: sequential integers, immutable once applied. Run `make migrate-up` after pull.
- **PII encryption**: `pgp_sym_encrypt(value, current_setting('app.enc_key'))` for email/phone in DB.
- **India DLT**: every outbound SMS must match a stored `dlt_templates.regex`; promotional window 10:00–21:00 IST.
- **SSE**: runs on port 8082 inside the same `cmd/server` binary (infinite read/write timeouts for streaming).

## Infrastructure ports (local dev)

| Service | Port |
|---|---|
| qeet-notify-server (API) | 8080 |
| qeet-notify-server (SSE) | 8082 |
| qeet-notify console | 3010 |
| PostgreSQL 17 | 5433 |
| NATS | 4222 / 8222 (monitor) |
| Redis 7 | 6379 |
| Prometheus metrics | 9090 |

## Deployment

See [deployments/](deployments/) — GHCR + SSH + `docker compose up -d`.  
Binaries: `server`, `worker`, `workflow`, `analytics`, `migrate`, `scheduler`.
