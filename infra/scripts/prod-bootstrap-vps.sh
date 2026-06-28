#!/usr/bin/env bash
# First deploy on a fresh VPS: Docker + /opt/qxsystem (run via GitHub Actions SSH).
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "Installing Docker..."
  curl -fsSL https://get.docker.com | sudo sh
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 plugin is required" >&2
  exit 1
fi

deploy_user="$(whoami)"
sudo mkdir -p /opt/qxsystem/nginx
sudo chown -R "${deploy_user}:${deploy_user}" /opt/qxsystem

if [[ "${deploy_user}" != "root" ]]; then
  sudo usermod -aG docker "${deploy_user}" 2>/dev/null || true
fi

echo "Bootstrap OK: /opt/qxsystem (user=${deploy_user})"
