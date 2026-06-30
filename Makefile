BINARY     := qeet-notify-api
MODULE     := github.com/qeetgroup/qeet-notify
GO         := go
GOFLAGS    ?=
BUILD_DIR  := bin

# DB / migrate
DB_URL     ?= postgres://qeet-notify:qeet-notify@localhost:5433/qeet-notify?sslmode=disable
MIGRATIONS := migrations

# Build info
GIT_SHA    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X main.version=$(GIT_SHA) -X main.buildTime=$(BUILD_TIME)

.PHONY: help install build dev dev-backend test test-backend test-integration \
        lint fmt vet db-up db-down db-reset migrate-up migrate-down migrate-force \
        seed clean kill

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

install: ## Install Go deps + JS deps
	$(GO) mod tidy
	@if [ -d apps/console ]; then cd apps/console && pnpm install; fi

# ── Build ──────────────────────────────────────────────────────────────────────

build: ## Build all Go binaries
	mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-server    ./cmd/server/
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-worker    ./cmd/worker/
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-workflow  ./cmd/workflow/
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-analytics ./cmd/analytics/
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-migrate   ./cmd/migrate/
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-scheduler ./cmd/scheduler/

build-api: ## Build API server only
	mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/qeet-notify-api ./cmd/server/

# ── Dev ────────────────────────────────────────────────────────────────────────

dev: ## Run API server in dev mode (hot reload requires air)
	DATABASE_URL=$(DB_URL) $(GO) run ./cmd/server/

dev-backend: dev ## Alias for dev

dev-console: ## Start Next.js console (:3010)
	cd apps/console && pnpm dev

# ── Test ───────────────────────────────────────────────────────────────────────

test: vet ## Run all Go unit tests
	$(GO) test -race -count=1 ./...

test-backend: test ## Alias for test

test-integration: ## Run integration tests (requires Docker)
	$(GO) test -race -count=1 -tags integration ./...

# ── Quality ────────────────────────────────────────────────────────────────────

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go code
	$(GO) fmt ./...

# ── Database ───────────────────────────────────────────────────────────────────

db-up: ## Start local PostgreSQL via Docker Compose
	docker compose up -d postgres
	@echo "Waiting for postgres..."; \
	until docker compose exec postgres pg_isready -q 2>/dev/null; do sleep 1; done
	@echo "PostgreSQL ready at localhost:5433"

db-down: ## Stop local PostgreSQL
	docker compose stop postgres

db-reset: ## Drop and recreate the local database
	docker compose exec postgres psql -U qeet-notify -c "DROP DATABASE IF EXISTS \"qeet-notify\";"
	docker compose exec postgres psql -U qeet-notify -c "CREATE DATABASE \"qeet-notify\";"
	$(MAKE) migrate-up

db-wipe: ## Remove PostgreSQL volume entirely
	docker compose down -v

db-psql: ## Open psql shell
	docker compose exec postgres psql -U qeet-notify -d qeet-notify

db-nats: ## Start NATS + Redis alongside Postgres
	docker compose up -d

# ── Migrations ─────────────────────────────────────────────────────────────────

migrate-up: ## Apply all pending migrations
	$(GO) run ./cmd/migrate/ -url "$(DB_URL)" -dir "$(MIGRATIONS)" up

migrate-down: ## Roll back last migration
	$(GO) run ./cmd/migrate/ -url "$(DB_URL)" -dir "$(MIGRATIONS)" down 1

migrate-force: ## Force migration version (V=<n>)
	$(GO) run ./cmd/migrate/ -url "$(DB_URL)" -dir "$(MIGRATIONS)" force $(V)

migrate-down-all: ## Roll back ALL migrations
	$(GO) run ./cmd/migrate/ -url "$(DB_URL)" -dir "$(MIGRATIONS)" down -all

migrate-version: ## Print current migration version
	$(GO) run ./cmd/migrate/ -url "$(DB_URL)" -dir "$(MIGRATIONS)" version

# ── Seed ───────────────────────────────────────────────────────────────────────

seed: ## Seed demo data
	DATABASE_URL=$(DB_URL) $(GO) run ./tools/seed/

# ── Infra ──────────────────────────────────────────────────────────────────────

infra-up: ## Start all local infra (Postgres, NATS, Redis, MinIO)
	docker compose up -d

infra-down: ## Stop all local infra
	docker compose down

# ── Utility ────────────────────────────────────────────────────────────────────

kill: ## Kill any process listening on :8080
	-lsof -ti:8080 | xargs kill -9 2>/dev/null || true

clean: ## Remove build artefacts
	rm -rf $(BUILD_DIR)
