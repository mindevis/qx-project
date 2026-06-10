.PHONY: dev-up dev-down api agent launcher web test lint jwt-secret jwt-secret-env

dev-up:
	docker compose -f infra/docker/docker-compose.yml up -d

dev-down:
	docker compose -f infra/docker/docker-compose.yml down

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
	cd services/qxlauncher && go build -o ../../bin/qx-launcher ./cmd

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
