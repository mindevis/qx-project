#!/usr/bin/env bash
# On VPS: pull GHCR images and restart stack (/opt/qxsystem).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT/.env.prod}"
TAG_FILE="$ROOT/image-tag.env"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE (should be deployed by GitHub Actions)." >&2
  exit 1
fi
if [[ ! -f "$TAG_FILE" ]]; then
  echo "Missing $TAG_FILE (should be deployed by GitHub Actions)." >&2
  exit 1
fi

COMPOSE="docker compose -f $ROOT/docker-compose.yml --env-file $ENV_FILE --env-file $TAG_FILE"

if [[ -n "${GHCR_TOKEN:-}" ]]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "${GHCR_USER:-github}" --password-stdin
fi

$COMPOSE pull api web
$COMPOSE up -d --no-build
$COMPOSE ps

echo "Stack up."
