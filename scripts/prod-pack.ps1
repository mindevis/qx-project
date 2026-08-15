#Requires -Version 5.1
$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$ComposeDir = Join-Path $Root "infra/docker"
$EnvFile = if ($env:ENV_FILE) { $env:ENV_FILE } else { Join-Path $ComposeDir ".env.prod" }
$Bundle = Join-Path $Root "dist/qx-prod"
$Tag = if ($env:QX_IMAGE_TAG) { $env:QX_IMAGE_TAG } else { "prod" }

if (-not (Test-Path $EnvFile)) {
    $BuildEnv = Join-Path $ComposeDir ".env.prod.build"
    if (Test-Path $BuildEnv) {
        $EnvFile = $BuildEnv
        Write-Host "Using build env: $EnvFile"
    } else {
        Write-Error "Missing .env.prod — copy from .env.prod.example"
    }
}

Set-Location $Root
Write-Host "Building qx-agent-linux..."
& make build-agent-linux

Write-Host "Building qx-launcher.exe..."
& make build-launcher-win

Write-Host "Building Docker images..."
docker compose -f (Join-Path $ComposeDir "docker-compose.prod.yml") --env-file $EnvFile build

New-Item -ItemType Directory -Force -Path (Join-Path $Bundle "images"), (Join-Path $Bundle "nginx") | Out-Null
Write-Host "Saving images..."
docker save "qx-api:$Tag" -o (Join-Path $Bundle "images/qx-api.tar")
docker save "qx-web:$Tag" -o (Join-Path $Bundle "images/qx-web.tar")

Copy-Item (Join-Path $ComposeDir "docker-compose.prod.runtime.yml") (Join-Path $Bundle "docker-compose.yml")
Copy-Item (Join-Path $ComposeDir "nginx/prod.conf") (Join-Path $Bundle "nginx/")
Copy-Item (Join-Path $ComposeDir "nginx/spa-security-headers.conf") (Join-Path $Bundle "nginx/")
Copy-Item (Join-Path $ComposeDir "nginx/gzip.conf") (Join-Path $Bundle "nginx/")
Copy-Item (Join-Path $ComposeDir "nginx/upstream-proxies.conf") (Join-Path $Bundle "nginx/")
Copy-Item (Join-Path $ComposeDir "nginx/qtranslator.conf") (Join-Path $Bundle "nginx/")
Copy-Item (Join-Path $ComposeDir ".env.prod.example") $Bundle
"QX_API_IMAGE=qx-api:$Tag" | Set-Content -Path (Join-Path $Bundle "image-tag.env") -Encoding utf8NoBOM
Add-Content -Path (Join-Path $Bundle "image-tag.env") -Value "QX_WEB_IMAGE=qx-web:$Tag" -Encoding utf8NoBOM
Copy-Item (Join-Path $Root "infra/scripts/prod-up.sh") (Join-Path $Bundle "up.sh")

Write-Host ""
Write-Host "Bundle ready: $Bundle"
Write-Host "  1. Copy dist/qx-prod/ to dedicated server (e.g. scp -r dist/qx-prod user@host:/opt/qx-prod)"
Write-Host "  2. On dedicated server: cp .env.prod.example .env.prod && edit secrets"
Write-Host "  3. On dedicated server: ./up.sh"
