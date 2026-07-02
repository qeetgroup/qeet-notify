#!/usr/bin/env bash
# migrate.sh — thin wrapper around golang-migrate.
# Usage: bash scripts/migrate.sh up
#        bash scripts/migrate.sh down 1
#        bash scripts/migrate.sh version
set -euo pipefail

DIRECTION="${1:-up}"
STEPS="${2:-}"
DATABASE_URL="${DATABASE_URL:-postgres://qeet-notify:qeet-notify@localhost:5433/qeet-notify?sslmode=disable}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-migrations}"

if [ "$DIRECTION" = "up" ]; then
  migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
elif [ "$DIRECTION" = "down" ]; then
  migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down "${STEPS:-1}"
elif [ "$DIRECTION" = "version" ]; then
  migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version
else
  echo "Usage: $0 up|down [n]|version" >&2
  exit 1
fi
