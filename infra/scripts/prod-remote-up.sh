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

# Preserve qTranslator vhost across QX releases (managed by /opt/qtranslator deploy).
QT_CONF="$ROOT/nginx/qtranslator.conf"
QT_CONF_LIVE="/opt/qtranslator/nginx/qtranslator.conf"
if [[ -f "$QT_CONF_LIVE" ]] && grep -q "qtranslator-api" "$QT_CONF_LIVE" 2>/dev/null; then
  cp "$QT_CONF_LIVE" "$QT_CONF"
  echo "Nginx: restored qTranslator vhost from $QT_CONF_LIVE"
elif [[ -f "$QT_CONF" ]] && grep -q "qtranslator-api" "$QT_CONF" 2>/dev/null; then
  echo "Nginx: keeping existing qTranslator vhost"
elif [[ ! -f "$QT_CONF" ]]; then
  cat > "$QT_CONF" <<'PLACEHOLDER'
# Placeholder — replaced by qTranslator deploy.
server {
    listen 80;
    server_name qt.qx-dev.ru;
    return 503;
}
PLACEHOLDER
  echo "Nginx: wrote qTranslator placeholder vhost"
fi

for f in spa-security-headers.conf upstream-proxies.conf gzip.conf qtranslator.conf; do
  if [[ ! -f "$ROOT/nginx/$f" ]]; then
    echo "Missing nginx/$f (deploy bundle incomplete)." >&2
    exit 1
  fi
done

if [[ -n "${GHCR_TOKEN:-}" ]]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "${GHCR_USER:-github}" --password-stdin
fi

$COMPOSE pull api web
# nginx resolves api/web at config load — restart after web/api recreate to pick up new container IPs.
$COMPOSE up -d api web
$COMPOSE run --rm --no-deps nginx nginx -t

$COMPOSE up -d --no-build
$COMPOSE restart nginx
$COMPOSE ps

echo "Stack up."
