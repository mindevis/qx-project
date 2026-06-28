#!/usr/bin/env bash
# Build prod images locally and pack a deploy bundle for VPS (no build tools on server).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_DIR="$ROOT/infra/docker"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.prod}"
BUNDLE="$ROOT/dist/qx-prod"
TAG="${QX_IMAGE_TAG:-prod}"

if [[ ! -f "$ENV_FILE" ]]; then
  if [[ -f "$COMPOSE_DIR/.env.prod" ]]; then
    ENV_FILE="$COMPOSE_DIR/.env.prod"
  elif [[ -f "$COMPOSE_DIR/.env.prod.build" ]]; then
    ENV_FILE="$COMPOSE_DIR/.env.prod.build"
    echo "Using build env: $ENV_FILE (runtime secrets go in .env.prod on VPS)"
  else
    echo "Missing env file — copy .env.prod.example to .env.prod or use .env.prod.build" >&2
    exit 1
  fi
fi

cd "$ROOT"
echo "Building qx-agent-linux (embedded in API image)..."
make build-agent-linux

echo "Building Docker images..."
docker compose -f "$COMPOSE_DIR/docker-compose.prod.yml" --env-file "$ENV_FILE" build

mkdir -p "$BUNDLE/images" "$BUNDLE/nginx"
echo "Saving images to $BUNDLE/images/..."
docker save "qx-api:$TAG" -o "$BUNDLE/images/qx-api.tar"
docker save "qx-web:$TAG" -o "$BUNDLE/images/qx-web.tar"

cp "$COMPOSE_DIR/docker-compose.prod.runtime.yml" "$BUNDLE/docker-compose.yml"
cp "$COMPOSE_DIR/nginx/prod-split.conf" "$BUNDLE/nginx/"
cp "$ROOT/docs/schema.sql" "$BUNDLE/"
cp "$COMPOSE_DIR/.env.prod.example" "$BUNDLE/.env.prod.example"
echo "QX_API_IMAGE=qx-api:$TAG" > "$BUNDLE/image-tag.env"
echo "QX_WEB_IMAGE=qx-web:$TAG" >> "$BUNDLE/image-tag.env"
cp "$ROOT/infra/scripts/prod-up.sh" "$BUNDLE/up.sh"
chmod +x "$BUNDLE/up.sh"

echo ""
echo "Bundle ready: $BUNDLE/"
echo "  1. Copy dist/qx-prod/ to VPS (e.g. /opt/qx-prod)"
echo "  2. cp .env.prod.example .env.prod && edit secrets"
echo "  3. ./up.sh"
