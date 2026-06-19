# Services

| Папка | Компонент | Статус |
| ------- | ----------- | -------- |
| [qxapi](./qxapi/) | QXApi — backend | Phase 0–2 🟡 |
| [qxlauncher](./qxlauncher/) | QXLauncher — tray | Phase 1 ✅ |
| [qxagent](./qxagent/) | QXAgent — BYOS | Phase 2 🟡 |

Каждый сервис — отдельный Go-модуль в [go.work](../go.work).

Общие контракты WSS: [pkg/protocol](../pkg/protocol/).

## QXApi (`services/qxapi/`)

```text
cmd/           main, run (bootstrap Gin)
internal/
  api/         handlers, router, middleware, JSON errors
  auth/        JWT (user, guest, device, agent), bcrypt
  agenthub/    in-memory WSS hub for agents
  config/      env (API_ADDR, DATABASE_DSN, JWT_*, CORS, SSH_MASTER_KEY)
  crypto/      AES-GCM for SSH private keys
  database/    GORM Open, migrate, Ping
  launcher/    device link, instances, launch-requests
  models/      User, Server, Agent, …
  servers/     CRUD, deploy, start/stop via agent hub
  testutil/    SQLite helpers для тестов
```

Запуск: `make api` или `cd services/qxapi && go run ./cmd` (слушает `:3000`).

**REST prefix:** `/api/v1` — auth, users, launcher, servers, health.

**Agent WSS:** `GET /agent/v1/connect` — Bearer agent JWT.

Тесты: `go test ./...`.

## QXLauncher (`services/qxlauncher/`)

Device register/link, tray loop, Mojang Vanilla download + JVM launch.

Env: `QX_API_BASE_URL`, `QX_DEVICE_ID`, `QX_LAUNCH_DRY_RUN=1`.

## QXAgent (`services/qxagent/`)

WSS client к QXApi; обрабатывает `cmd.server.start/stop`, шлёт heartbeat и `res.server.*`.

Env:

| Variable | Description |
| -------- | ----------- |
| `QX_AGENT_TOKEN` | JWT от deploy (required) |
| `QX_API_BASE_URL` | e.g. `http://localhost:3000/api/v1` |
| `QX_AGENT_WS_URL` | override WSS URL |
| `QX_AGENT_DRY_RUN=1` | не запускать java, только лог |
