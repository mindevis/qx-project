# QXProject

Minecraft ecosystem: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent** — каждый в своей папке.

Документация: [docs/architecture.md](docs/architecture.md) · [docs/mvp.md](docs/mvp.md)

## Требования

- Go 1.22+ (Go workspace)
- Node.js 20+
- Docker (MySQL, Redis, MinIO)
- [Go extension][go-ext] для отладки в Cursor / VS Code (Delve ставится автоматически)

[go-ext]: https://marketplace.visualstudio.com/items?itemName=golang.go

## Быстрый старт

```bash
cp .env.example .env
make jwt-secret-env   # или: make jwt-secret — только вывести в консоль
make dev-up

# терминал 1 — QXApi
make tidy
make api

# терминал 2 — QXWeb
cd web/qxweb && npm install && npm run dev

# терминал 3 — QXLauncher (Windows tray, опционально)
make launcher
```

- Web: [localhost:5173](http://localhost:5173)
- API base: [localhost:3000/api/v1](http://localhost:3000/api/v1)
- Health: [health](http://localhost:3000/api/v1/health) · Ready: [health/ready](http://localhost:3000/api/v1/health/ready)

### Игра (Flow A / B)

1. Запустите **QXLauncher** (`make launcher` или `make build-launcher`) — в консоли появится `link_url`, откройте его в браузере.
2. **Guest:** подтвердите связку на `/launcher/link` без логина.
3. **Registered:** войдите на сайте → `/launcher/link?device=…` → подтвердите связку.
4. На `/launcher` создайте Vanilla-инстанс и offline-профиль → **Играть** (launch-bridge → tray → JVM).

Переменные для dev: `QX_SKIP_TRAY=1` (без systray), `QX_LAUNCH_DRY_RUN=1` (без реального JVM), `QX_API_BASE_URL=http://localhost:3000/api/v1`.

### Сервер (Flow C)

1. Войдите на сайте → **Servers** → добавьте VPS (SSH).
2. **Deploy** (dry-run в dev без бинарника агента) → **Start/Stop** → live-консоль.

Подробнее: [docs/mvp.md](docs/mvp.md) · [docs/device-linking.md](docs/device-linking.md) · [docs/launch-bridge.md](docs/launch-bridge.md)

## Отладка (Cursor / VS Code)

1. `cp .env.example .env`
2. Run and Debug (`F5`) → выбрать конфигурацию из `.vscode/launch.json`:

| Конфигурация | Что делает |
| --- | --- |
| **QXApi** | API с breakpoints |
| **QXAgent** | Агент с breakpoints |
| **QXLauncher** | Лаунчер с breakpoints |
| **QXWeb** | Vite dev-server в терминале |
| **Go: текущий тест** | Отладка теста в открытом `*_test.go` |
| **Vitest: текущий файл** | Отладка открытого `*.test.ts(x)` |

Docker (MySQL, Redis, MinIO) поднимается отдельно: **Terminal → Run Task → Docker: dev-up** (перед **QXApi**, если контейнеры ещё не запущены).

Переменные окружения читаются из `.env` в корне репозитория.

Если отладчик Go не стартует (`cannot launch dlv dap`): перезапустите Cursor, затем `Ctrl+Shift+P` → **Go: Install/Update Tools** → отметьте `dlv` и `dlv-dap`. В проекте включён legacy-адаптер Delve для Windows.

## Тесты

```bash
make test              # Go + web unit tests
make test-coverage     # с отчётом покрытия (100%)
make e2e-alpha         # API Flow A/B/C + tray dry-run + Playwright (Phase Alpha automated)
make e2e-dry-run       # API Flow A/B/C + tray launch dry-run (без JVM)
make build-launcher-win # bin/qx-launcher.exe (Windows)
```

CI (`.github/workflows/ci.yml`): unit tests, Playwright, `e2e-dry-run`, артефакт `qx-launcher-windows-amd64`.

## Структура репозитория

```text
services/
  qxapi/          QXApi — REST + WebSocket (Go)
  qxagent/        QXAgent — BYOS daemon (Go)
  qxlauncher/     QXLauncher — Windows tray (Go)
web/
  qxweb/          QXWeb — React SPA (panel + /launcher + /servers)
pkg/
  mcmanifest/     Mojang manifest helpers
  protocol/       Agent ↔ API WSS types
infra/docker/     docker-compose dev stack
docs/             архитектура & ADR
go.work           Go workspace
```

Код сервиса **не смешивается**: у QXApi свой `internal/`, у QXAgent и QXLauncher — свои (по мере реализации).

## Реализовано (MVP alpha)

- [x] **Phase 0** — auth, profile, CI, 100% unit coverage
- [x] **Phase 1** — device link, instances, launch-bridge, QXLauncher tray + Vanilla
- [x] **Phase 2** — servers CRUD, SSH deploy (dry-run), agent WSS, web console
- [x] **Phase 3** — registered user device status, JWT refresh в tray
- [ ] **Phase Alpha** — manual E2E pass, bug bash ([test matrix](docs/qa/test-matrix.md), [FAQ](docs/faq.md))

### Prod (Tier 0, один VPS)

```bash
cp infra/docker/.env.prod.example infra/docker/.env.prod
# JWT, MySQL, SSH_MASTER_KEY — см. .env.prod.example
make prod-build
make prod-up
# → http://localhost:8080
```
