# Services

| Папка | Компонент | Статус |
| ------- | ----------- | -------- |
| [qxapi](./qxapi/) | QXApi — backend | Phase 0 ✅ |
| [qxlauncher](./qxlauncher/) | QXLauncher — tray | Phase 1 (stub) |
| [qxagent](./qxagent/) | QXAgent — BYOS | Phase 2 (stub) |

Каждый сервис — отдельный Go-модуль в [go.work](../go.work).

Общие контракты WSS: [pkg/protocol](../pkg/protocol/) (placeholder).

## QXApi (`services/qxapi/`)

```text
cmd/           main, run (bootstrap Gin)
internal/
  api/         handlers, router, middleware, JSON errors
  auth/        JWT, bcrypt, Register/Login Service
  config/      env (API_ADDR, DATABASE_DSN, JWT_*, CORS)
  database/    GORM Open, migrate, Ping
  models/      User
  testutil/    SQLite helpers для тестов
```

Запуск: `make api` или `cd services/qxapi && go run ./cmd` (слушает `:3000`).

**REST prefix:** `/api/v1` — auth, users, health (`/health`, `/health/ready`).

Тесты: `go test ./...` — 100% statement coverage.

## Stubs

`qxlauncher` и `qxagent` — только `cmd/` с сообщением «not implemented»; тесты покрывают `run()`.
