.PHONY: dev-up dev-down dev-vps-up dev-vps-down dev-vps-rm dev-vps-rm-data dev-vps-info dev-vps-sh api agent launcher web test lint ci jwt-secret ssh-master-key prod-secrets jwt-secret-config gen-tray-icons prod-build prod-pack prod-up prod-down e2e-manual e2e-manual-dry-run e2e-api-smoke e2e-dry-run e2e-jvm e2e-web e2e-alpha test-forge-client test-neoforge-client test-fabric-client test-quilt-client build-launcher build-launcher-win gen-launcher-win-resources build-launcher-win-debug build-agent-linux swagger docs-serve docs-build

PROD_ENV_FILE := infra/docker/.env.prod
PROD_COMPOSE := docker compose -f infra/docker/docker-compose.prod.yml --env-file $(PROD_ENV_FILE)

ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif

dev-up:
	docker compose -f infra/docker/docker-compose.yml up -d

dev-down:
	docker compose -f infra/docker/docker-compose.yml down

dev-vps-keys:
	cd scripts/gen-dev-vps-key && set GOWORK=off&& go run . -dir ../../infra/docker/vps-dev/keys

dev-vps-up:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-vps.ps1

dev-vps-down:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-vps.ps1 -Down

dev-vps-rm:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-vps.ps1 -Rm

dev-vps-rm-data:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-vps.ps1 -Rm -WipeData

dev-vps-info:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-vps.ps1 -Info

dev-vps-sh:
	docker exec -it qx-vps-dev bash

prod-build:
	make build-agent-linux
	make build-launcher-win
	$(PROD_COMPOSE) build

prod-pack:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/prod-pack.ps1
else
	bash infra/scripts/prod-pack.sh
endif

prod-up:
	$(PROD_COMPOSE) up -d --no-build --pull never

prod-down:
	$(PROD_COMPOSE) down

api:
	cd services/qxapi && go run ./cmd

agent:
	cd services/qxagent && go run ./cmd

launcher:
	cd services/qxlauncher && go run ./cmd

gen-tray-icons:
	cd scripts/gen-tray-icon && go run . ../../services/qxlauncher/internal/tray/assets

web:
	cd web/qxweb && npm run dev

test:
	cd services/qxapi && go test ./...
	cd services/qxagent && go test ./...
	cd services/qxlauncher && go test ./...
	cd pkg/log && go test ./...
	cd pkg/reporoot && go test ./...
	cd pkg/envfile && go test ./...
	cd pkg/mcmanifest && go test ./...
	cd pkg/protocol && go test ./...
	cd pkg/safepath && go test ./...
	cd pkg/mojangjava && go test ./...
	cd web/qxweb && npm test

test-coverage:
	cd services/qxapi && go test ./cmd/... -coverprofile=coverage.out && go tool cover -func coverage.out | findstr /C:"total:"
	cd services/qxagent && go test ./... -coverprofile=coverage.out && go tool cover -func coverage.out | findstr /C:"total:"
	cd services/qxlauncher && go test ./... -coverprofile=coverage.out && go tool cover -func coverage.out | findstr /C:"total:"
	cd web/qxweb && npm run test:coverage

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --config ../../.golangci.yml

lint:
	cd services/qxapi && go vet ./... && $(GOLANGCI_LINT) ./...
	cd services/qxagent && go vet ./... && $(GOLANGCI_LINT) ./...
	cd services/qxlauncher && go vet ./... && $(GOLANGCI_LINT) ./...
	cd web/qxweb && npm run lint

# Mirrors GitHub Actions CI (go + web lint/coverage + Windows launcher cross-build).
ci: lint test-coverage build-launcher-win

AGENT_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
AGENT_LDFLAGS = -X main.agentVersion=$(AGENT_VERSION)
# Systray GUI app: release Windows builds use the windowsgui subsystem (no console).
LAUNCHER_WIN_GUI_LDFLAGS = -H=windowsgui
LAUNCHER_PROD_API_BASE ?= https://mc.qx-dev.ru/api/v1
LAUNCHER_PROD_WEB_BASE ?= https://mc.qx-dev.ru
LAUNCHER_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
LAUNCHER_PROD_LDFLAGS = $(LAUNCHER_WIN_GUI_LDFLAGS) \
	-X github.com/qxproject/qx/services/qxlauncher/internal/config.embeddedAPIBaseURL=$(LAUNCHER_PROD_API_BASE) \
	-X github.com/qxproject/qx/services/qxlauncher/internal/config.embeddedWebBaseURL=$(LAUNCHER_PROD_WEB_BASE) \
	-X github.com/qxproject/qx/services/qxlauncher/internal/version.Version=$(LAUNCHER_VERSION)

build-api:
	cd services/qxapi && go build -o ../../bin/qx-api ./cmd

build-agent:
	cd services/qxagent && go build -ldflags "$(AGENT_LDFLAGS)" -o ../../bin/qx-agent ./cmd

ifeq ($(OS),Windows_NT)
build-agent-linux:
	cd services/qxagent && set GOOS=linux&& set GOARCH=amd64&& go build -ldflags "$(AGENT_LDFLAGS)" -o ../../bin/qx-agent-linux ./cmd
else
build-agent-linux:
	cd services/qxagent && GOOS=linux GOARCH=amd64 go build -ldflags "$(AGENT_LDFLAGS)" -o ../../bin/qx-agent-linux ./cmd
endif

ifeq ($(OS),Windows_NT)
build-launcher:
	cd services/qxlauncher && go build -ldflags "$(LAUNCHER_WIN_GUI_LDFLAGS)" -o ../../bin/qx-launcher$(EXE) ./cmd
else
build-launcher:
	cd services/qxlauncher && go build -o ../../bin/qx-launcher$(EXE) ./cmd
endif

gen-launcher-win-resources:
	cd services/qxlauncher && go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 \
		-64 \
		-manifest cmd/app.manifest \
		-icon internal/tray/assets/icon.ico \
		-o cmd/rsrc_windows_amd64.syso \
		versioninfo.json

ifeq ($(OS),Windows_NT)
build-launcher-win: gen-launcher-win-resources
	cd services/qxlauncher && go build -trimpath -ldflags "$(LAUNCHER_PROD_LDFLAGS)" -o ../../bin/qx-launcher.exe ./cmd
else
build-launcher-win: gen-launcher-win-resources
	cd services/qxlauncher && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LAUNCHER_PROD_LDFLAGS)" -o ../../bin/qx-launcher.exe ./cmd
endif

ifeq ($(OS),Windows_NT)
build-launcher-win-debug:
	cd services/qxlauncher && go build -o ../../bin/qx-launcher-debug.exe ./cmd
else
build-launcher-win-debug:
	cd services/qxlauncher && GOOS=windows GOARCH=amd64 go build -o ../../bin/qx-launcher-debug.exe ./cmd
endif

build-web:
	cd web/qxweb && npm run build

jwt-secret:
	cd scripts/gen-jwt-secret && go run .

ssh-master-key:
	cd scripts/gen-jwt-secret && go run . -ssh-master

prod-secrets:
	@echo PROD_JWT_SECRET:
	@$(MAKE) jwt-secret
	@echo.
	@echo PROD_SSH_MASTER_KEY:
	@$(MAKE) ssh-master-key

jwt-secret-config:
	cd scripts/gen-jwt-secret && go run . -toml ../../qxapi.toml

tidy:
	cd services/qxapi && go mod tidy
	cd services/qxagent && go mod tidy
	cd services/qxlauncher && go mod tidy

swagger:
	cd services/qxapi && go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g internal/api/openapi.go -o docs --parseInternal

e2e-manual:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/e2e-manual.ps1

e2e-manual-dry-run:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/e2e-manual.ps1 -RunDryRun

e2e-api-smoke:
	cd services/qxapi && go test ./internal/api -run "TestRouterFlow" -count=1

e2e-dry-run: e2e-api-smoke
	cd services/qxlauncher && go test ./internal/tray -run "TestRunLoop_DryLaunchOnce|TestExecuteLaunchDryRun" -count=1

e2e-jvm:
	cd pkg/mcmanifest && go test -run TestIntegrationMojangManifest1_21 -count=1
	cd services/qxlauncher && go test ./internal/minecraft -run TestIntegrationJavaOnPath -count=1

e2e-web:
	cd web/qxweb && npm run test:e2e:install && npm run test:e2e

e2e-alpha: e2e-api-smoke e2e-dry-run e2e-web

ifeq ($(OS),Windows_NT)
test-forge-client:
	cd services/qxlauncher && set QX_FORGE_E2E=1&& go test ./internal/minecraft -run TestIntegrationForgeClientLaunch -count=1 -timeout 30m -v
test-neoforge-client:
	cd services/qxlauncher && set QX_NEOFORGE_E2E=1&& go test ./internal/minecraft -run TestIntegrationNeoForgeClientLaunch -count=1 -timeout 30m -v
test-fabric-client:
	cd services/qxlauncher && set QX_FABRIC_E2E=1&& go test ./internal/minecraft -run TestIntegrationFabricClientLaunch -count=1 -timeout 30m -v
test-quilt-client:
	cd services/qxlauncher && set QX_QUILT_E2E=1&& go test ./internal/minecraft -run TestIntegrationQuiltClientLaunch -count=1 -timeout 30m -v
else
test-forge-client:
	cd services/qxlauncher && QX_FORGE_E2E=1 go test ./internal/minecraft -run TestIntegrationForgeClientLaunch -count=1 -timeout 30m -v
test-neoforge-client:
	cd services/qxlauncher && QX_NEOFORGE_E2E=1 go test ./internal/minecraft -run TestIntegrationNeoForgeClientLaunch -count=1 -timeout 30m -v
test-fabric-client:
	cd services/qxlauncher && QX_FABRIC_E2E=1 go test ./internal/minecraft -run TestIntegrationFabricClientLaunch -count=1 -timeout 30m -v
test-quilt-client:
	cd services/qxlauncher && QX_QUILT_E2E=1 go test ./internal/minecraft -run TestIntegrationQuiltClientLaunch -count=1 -timeout 30m -v
endif

docs-serve:
	python -m pip install -r docs/requirements.txt
	python -m mkdocs serve

docs-build:
	python -m pip install -r docs/requirements.txt
	python -m mkdocs build --strict
