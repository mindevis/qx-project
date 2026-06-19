#!/usr/bin/env bash
# QXProject - Phase Alpha manual E2E checklist (Linux/macOS)
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
echo "=== QXProject Alpha - manual E2E ==="
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
echo "  2. make launcher (or bin/qx-launcher) - open link_url from console"
echo "  3. Confirm link on /launcher/link (logged in)"
echo "  4. /launcher - create Vanilla instance + offline profile - Play"
echo "  5. Tray receives launch-request; JVM starts (or QX_LAUNCH_DRY_RUN=1)"
echo ""
echo "--- Flow B (guest) ---"
echo "  1. make launcher - open link_url without login"
echo "  2. Guest confirm on /launcher/link"
echo "  3. Create instance - Play (default nick Player)"
echo ""
echo "--- Flow C (server admin) ---"
echo "  1. Servers - add VPS (SSH) - Deploy (dry-run OK in dev)"
echo "  2. make agent (QX_AGENT_DRY_RUN=1) with token from deploy logs"
echo "  3. Start/Stop - live console"
echo ""
echo "--- Tray manual (test matrix A09, L03) ---"
echo "  Systray - Link launcher - browser opens /launcher/link"
echo "  Do NOT set QX_SKIP_TRAY=1 for this check."
echo ""
echo "Dev shortcuts:"
echo "  QX_SKIP_TRAY=1          - console-only launcher"
echo "  QX_LAUNCH_DRY_RUN=1     - skip real JVM"
echo "  QX_AGENT_DRY_RUN=1      - agent without JAR process"
echo ""
echo "Automated:"
echo "  make e2e-api-smoke      - API Flow A/B/C (router_test)"
echo "  make e2e-dry-run        - API smoke + tray launch-bridge dry-run"
echo ""

if [[ "$RUN_SMOKE" -eq 1 ]]; then
  echo "Running e2e-dry-run..."
  (cd "$ROOT" && make e2e-dry-run)
  echo "e2e-dry-run: OK"
  echo ""
fi
