#!/usr/bin/env bash
# Local smoke from repo (images must exist — run make prod-build first).
# For dedicated server: make prod-pack → copy dist/qx-prod/ → ./up.sh on server.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_DIR="$ROOT/infra/docker"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.prod}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE — copy from .env.prod.example and set secrets." >&2
  exit 1
fi

cd "$ROOT"
if ! docker image inspect qx-api:prod >/dev/null 2>&1; then
  echo "Images not found — run: make prod-build  (or make prod-pack for dedicated server bundle)" >&2
  exit 1
fi

cd "$COMPOSE_DIR"
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" up -d --no-build --pull never
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" ps

echo "Stack up. For dedicated server deploy use: make prod-pack"
