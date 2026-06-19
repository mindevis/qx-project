.PHONY: dev-up dev-down api agent launcher web test lint jwt-secret jwt-secret-env prod-build prod-up prod-down e2e-manual e2e-manual-dry-run e2e-api-smoke e2e-dry-run e2e-jvm e2e-web e2e-alpha build-launcher-win

ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif

dev-up:
	docker compose -f infra/docker/docker-compose.yml up -d

dev-down:
	docker compose -f infra/docker/docker-compose.yml down

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

jwt-secret-env:
	cd scripts/gen-jwt-secret && go run . -env ../../.env

tidy:
	cd services/qxapi && go mod tidy
	cd services/qxagent && go mod tidy
	cd services/qxlauncher && go mod tidy

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
