#!/usr/bin/env bash
# On dedicated server: TLS (optional Cloudflare DNS) → pull GHCR → restart stack (/opt/qxsystem).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT/.env.prod}"
TAG_FILE="$ROOT/image-tag.env"
DOMAIN="${CERTBOT_DOMAIN:-mc.qx-dev.ru}"
NGINX_ACTIVE="$ROOT/nginx/active.conf"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE (should be deployed by GitHub Actions)." >&2
  exit 1
fi
if [[ ! -f "$TAG_FILE" ]]; then
  echo "Missing $TAG_FILE (should be deployed by GitHub Actions)." >&2
  exit 1
fi

COMPOSE="docker compose -f $ROOT/docker-compose.yml --env-file $ENV_FILE --env-file $TAG_FILE"

cat "$ENV_FILE" "$TAG_FILE" > "$ROOT/.env"
chmod 600 "$ROOT/.env"

use_tls=false
if [[ -n "${CLOUDFLARE_API_TOKEN:-}" && -n "${CERTBOT_EMAIL:-}" ]]; then
  bash "$ROOT/certbot-cloudflare.sh"
  use_tls=true
elif [[ -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ]]; then
  use_tls=true
fi

if [[ "$use_tls" == "true" ]]; then
  cp "$ROOT/nginx/prod-tls.conf" "$NGINX_ACTIVE"
  echo "Nginx: TLS (Let's Encrypt)"
else
  cp "$ROOT/nginx/prod-http.conf" "$NGINX_ACTIVE"
  echo "Nginx: HTTP only (set PROD_CLOUDFLARE_API_TOKEN + PROD_CERTBOT_EMAIL for HTTPS)"
fi

if [[ -n "${GHCR_TOKEN:-}" ]]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "${GHCR_USER:-github}" --password-stdin
fi

$COMPOSE pull api web
$COMPOSE up -d --no-build
$COMPOSE ps

echo "Stack up."
