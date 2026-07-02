#!/usr/bin/env bash
# generate.sh — re-run all code generation.
# Currently: none. Add OpenAPI bundler, mockgen, sqlc, or protoc invocations here.
set -euo pipefail

echo "==> Bundling OpenAPI spec..."
# TODO: add a YAML bundler (e.g. redocly bundle) to merge api/openapi/v1/*.yaml
# into api/openapi/combined.yaml once the per-resource files are populated.
# Example: npx @redocly/cli bundle api/openapi/v1/_base.yaml --output api/openapi/combined.yaml

echo "==> No generators configured yet — add them to this script."
