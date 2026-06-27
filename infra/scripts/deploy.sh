#!/usr/bin/env bash
# Deploy QXSystem prod stack on a VPS (Docker Compose Tier 0).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_DIR="$ROOT/infra/docker"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.prod}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE — copy from .env.prod.example and set secrets." >&2
  exit 1
fi

cd "$ROOT"
if [[ ! -f bin/qx-agent-linux ]]; then
  echo "Building qx-agent-linux for SSH deploy..."
  make build-agent-linux
fi

cd "$COMPOSE_DIR"
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" build
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" up -d
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" ps

echo "Stack up. Check health: curl -fsS \"\${QX_PUBLIC_API_URL:-http://localhost:\${HTTP_PORT:-8080}}/api/v1/health\""
