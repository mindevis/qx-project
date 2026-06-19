#!/usr/bin/env bash
# Deploy QXProject prod stack on a VPS (Docker Compose Tier 0).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_DIR="$ROOT/infra/docker"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.prod}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE — copy from .env.prod.example and set secrets." >&2
  exit 1
fi

cd "$COMPOSE_DIR"
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" build
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" up -d
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" ps

echo "Stack up. Open http://localhost:\${HTTP_PORT:-8080} (or your VPS IP)."
