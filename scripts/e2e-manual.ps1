# QXProject - Phase Alpha manual E2E checklist (Windows)
# Usage: .\scripts\e2e-manual.ps1 [-RunSmoke]
# Prereq: Docker, Go, Node; copy .env.example to .env and run make jwt-secret-env

param(
    [switch]$RunSmoke,
    [switch]$RunDryRun,
    [switch]$RunJVM,
    [switch]$RunAll
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Write-Host ""
Write-Host "=== QXProject Alpha - manual E2E ===" -ForegroundColor Cyan
Write-Host "Docs: docs/qa/test-matrix.md, docs/mvp.md section 2 (DoD)"
Write-Host ""

function Test-Port([int]$Port) {
    try {
        $c = New-Object System.Net.Sockets.TcpClient
        $c.Connect("127.0.0.1", $Port)
        $c.Close()
        return $true
    } catch { return $false }
}

$checks = @(
    @{ Name = "Docker (MySQL 3306)"; Ok = (Test-Port 3306) },
    @{ Name = "QXApi (3000)"; Ok = (Test-Port 3000) },
    @{ Name = "QXWeb (5173)"; Ok = (Test-Port 5173) }
)

Write-Host "Stack status:" -ForegroundColor Yellow
foreach ($c in $checks) {
    $mark = if ($c.Ok) { "[OK]" } else { "[--]" }
    Write-Host ("  {0} {1}" -f $mark, $c.Name)
}

if (-not $checks[0].Ok) {
    Write-Host ""
    Write-Host "Start infra: make dev-up" -ForegroundColor DarkYellow
}
if (-not $checks[1].Ok) {
    Write-Host "Start API:   make api" -ForegroundColor DarkYellow
}
if (-not $checks[2].Ok) {
    Write-Host "Start Web:   make web" -ForegroundColor DarkYellow
}

Write-Host ""
Write-Host "--- Flow A (registered) ---" -ForegroundColor Green
Write-Host "  1. Register + login at http://localhost:5173"
Write-Host "  2. make launcher (or bin/qx-launcher) - copy link_url from console"
Write-Host "  3. Confirm link on /launcher/link (logged in)"
Write-Host "  4. /launcher - create Vanilla instance + offline profile - Play"
Write-Host "  5. QXLauncher получает launch-request; JVM starts (or QX_LAUNCH_DRY_RUN=1)"

Write-Host "--- Flow B (guest) ---" -ForegroundColor Green
Write-Host "  1. make launcher - open link_url without login"
Write-Host "  2. Guest confirm on /launcher/link"
Write-Host "  3. Create instance - Play (default nick Player)"

Write-Host "--- Flow C (server admin) ---" -ForegroundColor Green
Write-Host "  Dev VPS (Debian 13 + SSH): make dev-vps-up"
Write-Host "  SSH: localhost:2222, user root, key infra/docker/vps-dev/keys/dev_id_ed25519"
Write-Host "  .env: QX_PUBLIC_API_URL=http://host.docker.internal:3000"
Write-Host "        QX_PUBLIC_API_URL=http://host.docker.internal:3000 (make dev-vps-up builds agent binary)"
Write-Host "  1. Servers - add VPS (SSH creds from make dev-vps-info) - Deploy agent"
Write-Host "  2. Agent auto-starts in container via systemd after deploy (tag Agent)"
Write-Host "  3. MC offline until JAR started - Stop/Restart + console only when minecraft_running"
Write-Host "  Alt: POST /servers/{id}/start via API; QX_AGENT_DRY_RUN=1 on VPS for dry MC"

Write-Host "--- QXLauncher manual (test matrix A09, L03) ---" -ForegroundColor Green
Write-Host "  Icon in system tray - Link QXLauncher - browser opens /launcher/link"
Write-Host "  Do NOT set QX_SKIP_TRAY=1 for this check."

Write-Host ""
Write-Host "Dev shortcuts:" -ForegroundColor DarkGray
Write-Host "  QX_SKIP_TRAY=1          - console-only launcher"
Write-Host "  QX_LAUNCH_DRY_RUN=1     - skip real JVM"
Write-Host "  QX_AGENT_DRY_RUN=1      - agent without JAR process"
Write-Host ""
Write-Host "Automated API smoke (Flow A/B/C):" -ForegroundColor Yellow
Write-Host "  make e2e-api-smoke   - go test router Flow A/B/C against in-memory API"
Write-Host "  make e2e-web         - Playwright: Flow A + B + C (mock API)"
Write-Host "  make e2e-alpha       - API smoke + dry-run + Playwright (all automated E2E)"
Write-Host "  make e2e-dry-run     - API smoke + QXLauncher launch-bridge dry-run (I04 partial)"
Write-Host "  make e2e-jvm         - Mojang manifest + java -version (I04/I05 automated smoke)"
Write-Host ""
Write-Host "Full unit suite: make test"
Write-Host ""

if ($RunAll) {
    Write-Host "Running make e2e-alpha (API + dry-run + Playwright)..." -ForegroundColor Yellow
    Push-Location $root
    try {
        make e2e-alpha
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        Write-Host "e2e-alpha: OK" -ForegroundColor Green
        Write-Host ""
        Write-Host "Manual pass (A09, L03, I04, I05, Flow C): see docs/qa/test-matrix.md" -ForegroundColor Cyan
    } finally {
        Pop-Location
    }
} elseif ($RunJVM) {
    Write-Host "Running e2e-jvm (Mojang manifest + JVM smoke)..." -ForegroundColor Yellow
    Push-Location $root
    try {
        make e2e-jvm
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        Write-Host "e2e-jvm: OK" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next: manual QXLauncher (A09, L03) and full MC client launch (I04) — steps above." -ForegroundColor Cyan
    } finally {
        Pop-Location
    }
} elseif ($RunDryRun) {
    Write-Host "Running e2e-dry-run (API + QXLauncher dry launch)..." -ForegroundColor Yellow
    Push-Location $root
    try {
        make e2e-dry-run
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        Write-Host "e2e-dry-run: OK" -ForegroundColor Green
        Write-Host ""
    } finally {
        Pop-Location
    }
} elseif ($RunSmoke) {
    Write-Host "Running API smoke (Flow A/B/C)..." -ForegroundColor Yellow
    Push-Location $root
    try {
        make e2e-api-smoke
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        Write-Host "API smoke: OK" -ForegroundColor Green
        Write-Host ""
    } finally {
        Pop-Location
    }
}
