#!/usr/bin/env bash
# check-deps.sh — verify all required tools are installed and report versions.
set -euo pipefail

MISSING=0

check() {
  local name="$1"
  local cmd="$2"
  if command -v "$name" &>/dev/null; then
    echo "  OK  $name: $(eval "$cmd" 2>&1 | head -1)"
  else
    echo "  MISSING: $name"
    MISSING=$((MISSING + 1))
  fi
}

echo "==> Required tools"
check go        "go version"
check docker    "docker --version"
check migrate   "migrate -version"
check golangci-lint "golangci-lint --version"

echo ""
echo "==> Optional tools"
check node      "node --version"
check pnpm      "pnpm --version"
check helm      "helm version --short"
check kubectl   "kubectl version --client --short 2>/dev/null"

if [ $MISSING -gt 0 ]; then
  echo ""
  echo "Install missing tools before running 'make dev'."
  exit 1
fi
echo ""
echo "All required tools present."
