# Services

| Папка | Компонент | Статус |
| ------- | ----------- | -------- |
| [qxapi](./qxapi/) | QXApi — backend | Phase 0–3 ✅ |
| [qxlauncher](./qxlauncher/) | QXLauncher | Phase 1 ✅ |
| [qxagent](./qxagent/) | QXAgent — BYOS | Phase 2 ✅ |

**Prod deploy:** 🔲 — [mvp §7.1](../docs/mvp.md)

Каждый сервис — отдельный Go-модуль в [go.work](../go.work).

Общие контракты WSS: [pkg/protocol](../pkg/protocol/).

## QXApi (`services/qxapi/`)

```text
cmd/           main, run (bootstrap Gin)
internal/
  api/         handlers, router, middleware, JSON errors
  auth/        JWT (user, guest, device, agent), bcrypt
  agenthub/    in-memory WSS hub for agents
  config/      qxapi.toml at repo root
  crypto/      AES-GCM for SSH private keys
  database/    GORM Open, migrate, Ping
  launcher/    device link, instances, launch-requests
  models/      User, Server, Agent, …
  servers/     CRUD, deploy, start/stop via agent hub (`agent_online` / `minecraft_running`)
  testutil/    SQLite helpers для тестов
```

Запуск: `make api` или `cd services/qxapi && go run ./cmd` (слушает `:3000`).

**REST prefix:** `/api/v1` — auth, users, launcher, servers, health.

**Swagger UI:** `GET /swagger/index.html` (OpenAPI 3; regenerate: `make swagger` from repo root).

**Agent WSS:** `GET /agent/v1/connect` — Bearer agent JWT.

**Config:** [configuration.md](../docs/configuration.md) · [qxapi.toml.example](../qxapi.toml.example) · [launcher.toml.example](../launcher.toml.example) · [agent.toml.example](../agent.toml.example)

Тесты: `go test ./...`.

## QXLauncher (`services/qxlauncher/`)

Device register/link (HWID ПК, auto-open browser), launch-bridge poll, Mojang Vanilla download + JVM launch.

См. [device-linking.md](../docs/device-linking.md) · [configuration.md](../docs/configuration.md) (`device_id`, `web_base_url`).

**Config:** [launcher.toml.example](../launcher.toml.example) → `launcher.toml` (repo root dev; `~/.qxlauncher/` when installed). См. [configuration.md](../docs/configuration.md).

## QXAgent (`services/qxagent/`)

WSS client к QXApi; обрабатывает `cmd.server.start/stop`, шлёт heartbeat и `res.server.*`.

**Prod:** `/etc/qx-agent/agent.toml` (записывается при SSH deploy).

**Local dev:** [agent.toml.example](../agent.toml.example) → `agent.toml` в корне репо, или `-config /path/to/agent.toml`.

| Key (`agent.toml`) | Description |
| -------- | ----------- |
| `agent_token` | JWT от deploy |
| `api_base_url` | e.g. `http://localhost:3000/api/v1` |
| `ws_url` | override WSS URL |
| `dry_run` | не запускать java, только лог |
