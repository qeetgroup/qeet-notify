#!/usr/bin/env bash
# setup-dev.sh — one-shot dev environment bootstrap.
# Run once after cloning: bash scripts/setup-dev.sh
set -euo pipefail

echo "==> Checking Go version..."
go version

echo "==> Checking required tools..."
for tool in docker golangci-lint migrate; do
  if ! command -v "$tool" &>/dev/null; then
    echo "  MISSING: $tool (install it before continuing)"
  else
    echo "  OK: $tool"
  fi
done

echo "==> Copying .env if missing..."
if [ ! -f .env ]; then
  cp .env.example .env
  echo "  Created .env — fill in provider keys before running workers."
fi

echo "==> Starting dev infrastructure..."
docker compose -f deployments/docker-compose.dev.yml up -d

echo "==> Waiting for Postgres to be ready..."
until docker compose -f deployments/docker-compose.dev.yml exec -T postgres pg_isready -U qeet-notify &>/dev/null; do
  sleep 1
done

echo "==> Running migrations..."
bash scripts/migrate.sh up

echo ""
echo "Dev environment ready."
echo "  make dev            → start API server with live reload"
echo "  make dev-console    → start TanStack console on :3010"
