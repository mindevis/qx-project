#!/usr/bin/env bash
# Render .env.prod from environment (GitHub Actions secrets → deploy bundle).
set -euo pipefail

out="${1:?output path}"
: "${PROD_JWT_SECRET:?PROD_JWT_SECRET required}"
: "${PROD_SSH_MASTER_KEY:?PROD_SSH_MASTER_KEY required}"
: "${PROD_MYSQL_ROOT_PASSWORD:?PROD_MYSQL_ROOT_PASSWORD required}"
: "${PROD_MYSQL_PASSWORD:?PROD_MYSQL_PASSWORD required}"
: "${PROD_MINIO_PASSWORD:?PROD_MINIO_PASSWORD required}"

cors="${CORS_ORIGIN:-https://mc.qx-dev.ru}"
api_url="${QX_PUBLIC_API_URL:-https://mc.qx-dev.ru}"
mysql_db="${MYSQL_DATABASE:-qx}"
mysql_user="${MYSQL_USER:-qx}"

: > "$out"
kv() { printf '%s=%s\n' "$1" "$2" >> "$out"; }

kv HTTP_PORT "${HTTP_PORT:-80}"
kv HTTPS_PORT "${HTTPS_PORT:-443}"
kv CORS_ORIGIN "$cors"
kv QX_PUBLIC_API_URL "$api_url"
kv MYSQL_ROOT_PASSWORD "$PROD_MYSQL_ROOT_PASSWORD"
kv MYSQL_DATABASE "$mysql_db"
kv MYSQL_USER "$mysql_user"
kv MYSQL_PASSWORD "$PROD_MYSQL_PASSWORD"
kv DATABASE_DSN "${mysql_user}:${PROD_MYSQL_PASSWORD}@tcp(mysql:3306)/${mysql_db}?charset=utf8mb4&parseTime=True&loc=Local"
kv JWT_SECRET "$PROD_JWT_SECRET"
kv ACCESS_TOKEN_TTL "${ACCESS_TOKEN_TTL:-1h}"
kv REFRESH_TOKEN_TTL "${REFRESH_TOKEN_TTL:-168h}"
kv SSH_MASTER_KEY "$PROD_SSH_MASTER_KEY"
kv MINIO_ROOT_USER "${MINIO_ROOT_USER:-minio}"
kv MINIO_PASSWORD "$PROD_MINIO_PASSWORD"
kv MINIO_ROOT_PASSWORD "$PROD_MINIO_PASSWORD"
kv LOG_LEVEL "${LOG_LEVEL:-info}"
kv LOG_FORMAT "${LOG_FORMAT:-json}"
kv QX_AGENT_BINARY_PATH "/opt/qxsystem/bin/qx-agent-linux"

chmod 600 "$out"
