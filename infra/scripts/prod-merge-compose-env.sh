#!/usr/bin/env bash
# Merge .env.prod + image-tag.env → .env for plain `docker compose` on VPS.
set -euo pipefail

ROOT="${1:-/opt/qxsystem}"
ENV_PROD="$ROOT/.env.prod"
TAG_FILE="$ROOT/image-tag.env"
OUT="$ROOT/.env"

[[ -f "$ENV_PROD" ]] || { echo "missing $ENV_PROD" >&2; exit 1; }
[[ -f "$TAG_FILE" ]] || { echo "missing $TAG_FILE" >&2; exit 1; }

cat "$ENV_PROD" "$TAG_FILE" > "$OUT"
chmod 600 "$OUT"
