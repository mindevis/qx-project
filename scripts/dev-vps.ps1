# Dev Linux dedicated server for Flow C (Debian 13 + SSH + systemd).
# Usage: .\scripts\dev-vps.ps1          - start
#        .\scripts\dev-vps.ps1 -Down    - stop
#        .\scripts\dev-vps.ps1 -Rm      - remove container, volumes, local image
#        .\scripts\dev-vps.ps1 -Info    - print SSH creds

param(
    [switch]$Down,
    [switch]$Rm,
    [switch]$WipeData,
    [switch]$Info
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$keysDir = Join-Path $root "infra\docker\vps-dev\keys"
$privKey = Join-Path $keysDir "dev_id_ed25519"
$pubKey = "$privKey.pub"
$authKeys = Join-Path $keysDir "authorized_keys"
$composeBase = Join-Path $root "infra\docker\docker-compose.yml"
$composeVps = Join-Path $root "infra\docker\docker-compose.vps-dev.yml"

function Ensure-DevKeys {
    if (Test-Path $privKey) { return }
    New-Item -ItemType Directory -Force -Path $keysDir | Out-Null
    Write-Host "Generating dev SSH key: $privKey" -ForegroundColor Yellow
    $genDir = Join-Path $root "scripts\gen-dev-vps-key"
    Push-Location $genDir
    try {
        $env:GOWORK = "off"
        go run . -dir (Join-Path $root "infra\docker\vps-dev\keys")
        if ($LASTEXITCODE -ne 0) { throw "go run gen-dev-vps-key failed" }
    } finally {
        Pop-Location
    }
}

function Show-Info {
    $mcStart = if ($env:DEV_VPS_MC_PORT_START) { $env:DEV_VPS_MC_PORT_START } else { "25565" }
    $mcEnd = if ($env:DEV_VPS_MC_PORT_END) { $env:DEV_VPS_MC_PORT_END } else { "25585" }
    $rconStart = if ($env:DEV_VPS_RCON_PORT_START) { $env:DEV_VPS_RCON_PORT_START } else { "35565" }
    $rconEnd = if ($env:DEV_VPS_RCON_PORT_END) { $env:DEV_VPS_RCON_PORT_END } else { "35585" }

    Write-Host ""
    Write-Host "=== QX dev dedicated server (Flow C) ===" -ForegroundColor Cyan
    Write-Host "Host:     localhost"
    Write-Host "Port:     2222"
    Write-Host "User:     root"
    Write-Host "Key file: $privKey"
    Write-Host ""
    Write-Host "Minecraft (host -> container):"
    Write-Host "  Game ports: $mcStart-$mcEnd"
    Write-Host "  RCON ports: $rconStart-$rconEnd"
    Write-Host "  Client connect: localhost:<game-port>  (address in panel: localhost for dev)"
    Write-Host ""
    Write-Host "Data persists across container rebuilds (Docker volumes vps_dev_qx_data, vps_dev_agent_config)."
    Write-Host "Wipe game servers + agent: make dev-vps-rm-data"
    Write-Host ""
    Write-Host "Add to qxapi.toml (API restart required):"
    Write-Host "  public_api_url = `"http://host.docker.internal:3000`""
    Write-Host "  # agent_binary_path auto-detected after make build-agent-linux"
    Write-Host ""
    Write-Host "Panel: Servers -> Add server -> paste private key from key file -> Deploy"
    Write-Host "Test SSH: ssh -i `"$privKey`" -p 2222 -o StrictHostKeyChecking=no root@localhost"
    Write-Host ""
    Write-Host "After deploy, agent connects to API via host.docker.internal."
    Write-Host ""
}

Push-Location $root
try {
    if ($Info) {
        Ensure-DevKeys
        Show-Info
        exit 0
    }

    if ($Down) {
        docker compose -f $composeBase -f $composeVps stop vps-dev
        Write-Host "vps-dev stopped" -ForegroundColor Green
        exit 0
    }

    if ($Rm) {
        Write-Host "Removing qx-vps-dev (container + local image)..." -ForegroundColor Yellow
        $prevEap = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            docker compose -f $composeBase -f $composeVps stop vps-dev 2>&1 | Out-Null
            docker compose -f $composeBase -f $composeVps rm -sf vps-dev 2>&1 | Out-Null
            docker rmi docker-vps-dev:latest -f 2>&1 | Out-Null
        } finally {
            $ErrorActionPreference = $prevEap
        }
        if ($WipeData) {
            Write-Host "Removing dev dedicated server data volumes..." -ForegroundColor Yellow
            docker volume rm docker_vps_dev_qx_data docker_vps_dev_agent_config 2>&1 | Out-Null
            Write-Host "dev-vps removed - container and data wiped" -ForegroundColor Green
        } else {
            Write-Host "dev-vps removed - container/image gone; /opt/qxsystem data kept in Docker volumes" -ForegroundColor Green
            Write-Host "Wipe volumes: make dev-vps-rm-data" -ForegroundColor DarkGray
        }
        Write-Host "SSH keys kept at: $keysDir" -ForegroundColor DarkGray
        Write-Host "Run make dev-vps-up for a fresh dedicated server container" -ForegroundColor DarkGray
        exit 0
    }

    Ensure-DevKeys

    Write-Host "Building Linux agent binary..." -ForegroundColor Yellow
    & make build-agent-linux
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "Starting qx-vps-dev (Debian 13 + SSH)..." -ForegroundColor Yellow
    docker compose -f $composeBase -f $composeVps up -d --build vps-dev
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "Waiting for SSH on :2222..." -ForegroundColor DarkGray
    $deadline = (Get-Date).AddSeconds(45)
    $ready = $false
    while ((Get-Date) -lt $deadline) {
        try {
            $tcp = New-Object System.Net.Sockets.TcpClient
            $tcp.Connect("127.0.0.1", 2222)
            $tcp.Close()
            $ready = $true
            break
        } catch {
            Start-Sleep -Seconds 2
        }
    }
    if (-not $ready) {
        Write-Host "SSH port 2222 not ready yet - check: docker logs qx-vps-dev" -ForegroundColor DarkYellow
    } else {
        Write-Host "SSH ready on localhost:2222" -ForegroundColor Green
    }

    Show-Info
} finally {
    Pop-Location
}
