.PHONY: dev-up dev-down dev-vps-up dev-vps-down dev-vps-info api agent launcher web test lint jwt-secret jwt-secret-config prod-build prod-up prod-down e2e-manual e2e-manual-dry-run e2e-api-smoke e2e-dry-run e2e-jvm e2e-web e2e-alpha build-launcher-win build-agent-linux swagger

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

dev-vps-info:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-vps.ps1 -Info

prod-build:
	docker compose -f infra/docker/docker-compose.prod.yml build

prod-up:
	docker compose -f infra/docker/docker-compose.prod.yml --env-file infra/docker/.env.prod up -d

prod-down:
	docker compose -f infra/docker/docker-compose.prod.yml --env-file infra/docker/.env.prod down

api:
	cd services/qxapi && go run ./cmd

agent:
	cd services/qxagent && go run ./cmd

launcher:
	cd services/qxlauncher && go run ./cmd

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
	cd web/qxweb && npm test

test-coverage:
	cd services/qxapi && go test ./... -coverprofile=coverage.out && go tool cover -func coverage.out | findstr /C:"total:"
	cd services/qxagent && go test ./... -coverprofile=coverage.out && go tool cover -func coverage.out | findstr /C:"total:"
	cd services/qxlauncher && go test ./... -coverprofile=coverage.out && go tool cover -func coverage.out | findstr /C:"total:"
	cd web/qxweb && npm run test:coverage

lint:
	cd services/qxapi && go vet ./...
	cd services/qxagent && go vet ./...
	cd services/qxlauncher && go vet ./...
	cd web/qxweb && npm run lint

build-api:
	cd services/qxapi && go build -o ../../bin/qx-api ./cmd

build-agent:
	cd services/qxagent && go build -o ../../bin/qx-agent ./cmd

ifeq ($(OS),Windows_NT)
build-agent-linux:
	cd services/qxagent && set GOOS=linux&& set GOARCH=amd64&& go build -o ../../bin/qx-agent-linux ./cmd
else
build-agent-linux:
	cd services/qxagent && GOOS=linux GOARCH=amd64 go build -o ../../bin/qx-agent-linux ./cmd
endif

build-launcher:
	cd services/qxlauncher && go build -o ../../bin/qx-launcher$(EXE) ./cmd

ifeq ($(OS),Windows_NT)
build-launcher-win:
	cd services/qxlauncher && go build -o ../../bin/qx-launcher.exe ./cmd
else
build-launcher-win:
	cd services/qxlauncher && GOOS=windows GOARCH=amd64 go build -o ../../bin/qx-launcher.exe ./cmd
endif

build-web:
	cd web/qxweb && npm run build

jwt-secret:
	cd scripts/gen-jwt-secret && go run .

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
