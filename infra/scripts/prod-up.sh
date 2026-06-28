#!/usr/bin/env bash
# Local bundle: docker load + up (offline / dev smoke).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT/.env.prod}"
TAG_FILE="$ROOT/image-tag.env"
COMPOSE="docker compose -f $ROOT/docker-compose.yml --env-file $ENV_FILE"
if [[ -f "$TAG_FILE" ]]; then
  COMPOSE="$COMPOSE --env-file $TAG_FILE"
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE — copy from .env.prod.example and set secrets." >&2
  exit 1
fi
if [[ ! -f "$TAG_FILE" ]]; then
  echo "Missing $TAG_FILE — run make prod-pack first." >&2
  exit 1
fi

for tar in "$ROOT/images/qx-api.tar" "$ROOT/images/qx-web.tar"; do
  if [[ ! -f "$tar" ]]; then
    echo "Missing $tar — run make prod-pack" >&2
    exit 1
  fi
  echo "Loading $(basename "$tar")..."
  docker load -i "$tar"
done

$COMPOSE up -d --no-build --pull never
$COMPOSE ps

echo "Stack up."
