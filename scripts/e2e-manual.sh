#!/usr/bin/env bash
# QXSystem - Phase Alpha manual E2E checklist (Linux/macOS)
# Usage: ./scripts/e2e-manual.sh [--smoke]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RUN_SMOKE=0
if [[ "${1:-}" == "--smoke" ]]; then
  RUN_SMOKE=1
fi

port_open() {
  (echo >/dev/tcp/127.0.0.1/"$1") >/dev/null 2>&1
}

echo ""
echo "=== QXSystem Alpha - manual E2E ==="
echo "Docs: docs/qa/test-matrix.md, docs/mvp.md section 2 (DoD)"
echo ""

check_port() {
  if port_open "$1"; then
    echo "  [OK] $2"
  else
    echo "  [--] $2"
  fi
}

check_port 3306 "Docker (MySQL 3306)"
check_port 3000 "QXApi (3000)"
check_port 5173 "QXWeb (5173)"

echo ""
echo "--- Flow A (registered) ---"
echo "  1. Register + login at http://localhost:5173"
echo "  2. make launcher - browser opens /launcher/link?device=<HWID> automatically"
echo "  3. Confirm link on opened page (logged in) - «Связать устройство»"
echo "  4. /launcher - create Vanilla instance + offline profile - Play"
echo "  5. QXLauncher receives launch-request; JVM starts (or launch_dry_run in launcher.toml)"
echo ""
echo "--- Flow B (guest) ---"
echo "  1. make launcher - browser opens link page automatically"
echo "  2. «Продолжить как гость» on /launcher/link"
echo "  3. Create instance - Play (default nick Player)"
echo ""
echo "--- Flow C (server admin) ---"
echo "  dev dedicated server: make dev-vps-up (SSH localhost:2222)"
echo "  qxapi.toml: public_api_url = \"http://host.docker.internal:3000\""
echo "  1. Servers - add dedicated server (SSH) - Deploy agent"
echo "  2. Tag Agent in panel (agent_online); MC offline until JAR started"
echo "  3. Stop/Restart + console only when minecraft_running"
echo "  Alt: POST /servers/{id}/start via API"
echo ""
echo "--- QXLauncher manual (test matrix A09, L03) ---"
echo "  Icon in system tray - Link QXLauncher - browser opens /launcher/link"
echo "  Do NOT enable skip_tray in launcher.toml for this check."
echo ""
echo "Dev shortcuts (launcher.toml / agent.toml):"
echo "  skip_tray = true        - console-only QXLauncher"
echo "  launch_dry_run = true - skip real JVM"
echo "  dry_run = true        - agent without JAR"
echo ""
echo "Automated:"
echo "  make e2e-api-smoke      - API Flow A/B/C (router_test)"
echo "  make e2e-dry-run        - API smoke + QXLauncher launch-bridge dry-run"
echo ""

if [[ "$RUN_SMOKE" -eq 1 ]]; then
  echo "Running e2e-dry-run..."
  (cd "$ROOT" && make e2e-dry-run)
  echo "e2e-dry-run: OK"
  echo ""
fi
