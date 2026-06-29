#!/usr/bin/env bash
# Issue or renew Let's Encrypt cert via Cloudflare DNS-01 (no open port 80 required).
set -euo pipefail

DOMAIN="${CERTBOT_DOMAIN:-mc.qx-dev.ru}"
EMAIL="${CERTBOT_EMAIL:?set CERTBOT_EMAIL (Lets Encrypt account)}"
TOKEN="${CLOUDFLARE_API_TOKEN:?set CLOUDFLARE_API_TOKEN}"
CRED_FILE="${CLOUDFLARE_CREDENTIALS_FILE:-/opt/qxsystem/.cloudflare-credentials.ini}"
PROPAGATION="${CLOUDFLARE_PROPAGATION_SECONDS:-30}"

cert_path="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"

install_renewal_hook() {
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/deploy
  sudo tee /etc/letsencrypt/renewal-hooks/deploy/qx-reload-nginx.sh >/dev/null <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail
if [[ -f /opt/qxsystem/docker-compose.yml ]]; then
  docker compose -f /opt/qxsystem/docker-compose.yml exec -T nginx nginx -s reload 2>/dev/null || true
fi
HOOK
  sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/qx-reload-nginx.sh
}

if [[ -f "$cert_path" ]] && [[ "${FORCE_CERTBOT:-0}" != "1" ]]; then
  echo "Certificate already present: $cert_path"
  install_renewal_hook
  exit 0
fi

if ! command -v certbot >/dev/null 2>&1; then
  echo "Installing certbot + dns-cloudflare plugin…"
  sudo apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y certbot python3-certbot-dns-cloudflare
fi

sudo mkdir -p "$(dirname "$CRED_FILE")"
umask 077
printf 'dns_cloudflare_api_token = %s\n' "$TOKEN" | sudo tee "$CRED_FILE" >/dev/null
sudo chmod 600 "$CRED_FILE"

echo "Requesting certificate for ${DOMAIN} (DNS-01 / Cloudflare)…"
sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials "$CRED_FILE" \
  --dns-cloudflare-propagation-seconds "$PROPAGATION" \
  -d "$DOMAIN" \
  --email "$EMAIL" \
  --agree-tos \
  --non-interactive \
  --keep-until-expiring

install_renewal_hook

echo "Certificate OK: $cert_path"
