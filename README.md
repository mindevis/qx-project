# QXProject

Minecraft ecosystem: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent** — каждый в своей папке.

**Статус:** MVP alpha (dev) ✅ · Prod 🔲

Документация: [architecture](docs/architecture.md) · [mvp](docs/mvp.md) · [FAQ](docs/faq.md) · [test matrix](docs/qa/test-matrix.md)

| Документ | Содержание |
| -------- | ------------ |
| [mvp.md](docs/mvp.md) | Scope, DoD, фазы, prod readiness |
| [architecture.md](docs/architecture.md) | Полная архитектура + статус |
| [api.md](docs/api.md) | REST + WebSocket |
| [agent-protocol.md](docs/agent-protocol.md) | Agent WSS, deploy |
| [device-linking.md](docs/device-linking.md) | Launcher ↔ сайт |
| [launch-bridge.md](docs/launch-bridge.md) | Site → QXLauncher → JVM |
| [ssh-deploy.md](docs/ssh-deploy.md) | SSH deploy agent |
| [faq.md](docs/faq.md) | FAQ alpha |
| [adr/](docs/adr/) | Architecture Decision Records |

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

# терминал 3 — QXLauncher (опционально)
make launcher
```

- Web: [localhost:5173](http://localhost:5173)
- API base: [localhost:3000/api/v1](http://localhost:3000/api/v1)
- Health: [health](http://localhost:3000/api/v1/health) · Ready: [health/ready](http://localhost:3000/api/v1/health/ready)

### Игра (Flow A / B)

1. Запустите **QXLauncher** (`make launcher` или `make build-launcher`) — в консоли появится `link_url`, откройте его в браузере.
2. **Guest:** подтвердите связку на `/launcher/link` без логина.
3. **Registered:** войдите на сайте → `/launcher/link?device=…` → подтвердите связку.
4. На `/launcher` создайте Vanilla-инстанс и offline-профиль → **Играть** (launch-bridge → QXLauncher → JVM).

Переменные для dev: `QX_API_BASE_URL=http://localhost:3000/api/v1`, `QX_WEB_BASE_URL=http://localhost:5173`.

### Сервер (Flow C)

**Dev VPS (Debian 13 + SSH + systemd):**

```bash
make dev-vps-up      # контейнер qx-vps-dev, SSH :2222, ключ в infra/docker/vps-dev/keys/
make dev-vps-info    # host/port/user/key + подсказки для .env
```

В `.env` для реального SSH deploy (перезапустите API):

```env
QX_PUBLIC_API_URL=http://host.docker.internal:3000
```

Agent binary (`bin/qx-agent-linux`) собирается через `make dev-vps-up` и подхватывается API автоматически.

1. **Servers** → Add server: `localhost`, port `2222`, user `root`, private key из `infra/docker/vps-dev/keys/dev_id_ed25519`
2. **Deploy agent** → agent ставится в контейнер через SSH + systemd; в panel — тег **Agent** (`agent_online`)
3. **Minecraft** — JAR на VPS вручную (или post-MVP install pipeline); **Stop/Restart** и live-консоль в UI — только когда `minecraft_running`

Проверка SSH: `ssh -i infra/docker/vps-dev/keys/dev_id_ed25519 -p 2222 -o StrictHostKeyChecking=no root@localhost`

Подробнее: [docs/mvp.md](docs/mvp.md) · [docs/device-linking.md](docs/device-linking.md) · [docs/launch-bridge.md](docs/launch-bridge.md)

## Отладка (Cursor / VS Code)

1. `cp .env.example .env`
2. Run and Debug (`F5`) → выбрать конфигурацию из `.vscode/launch.json`:

| Конфигурация | Что делает |
| --- | --- |
| **QXApi** | API с breakpoints |
| **QXLauncher** | Лаунчер с breakpoints |
| **QXWeb** | Vite dev-server в терминале |
| **Dev VPS: up** | Flow C: Debian SSH на `:2222`, сборка `qx-agent-linux` |
| **Dev VPS: down** | Остановить контейнер `qx-vps-dev` |
| **Dev VPS: info** | SSH host/port и подсказки для `.env` |
| **QX Dev Stack** | QXApi + QXWeb + QXLauncher (compound) |
| **Go: текущий тест** | Отладка теста в открытом `*_test.go` |
| **Vitest: текущий файл** | Отладка открытого `*.test.ts(x)` |

Docker (MySQL, Redis, MinIO): **Terminal → Run Task → Docker: dev-up** (перед **QXApi**).

Flow C (серверы): **F5 → Dev VPS: up**, затем **QXApi**. В `.env`: `QX_PUBLIC_API_URL=http://host.docker.internal:3000`.

Переменные окружения читаются из `.env` в корне репозитория.

Если отладчик Go не стартует (`cannot launch dlv dap`): перезапустите Cursor, затем `Ctrl+Shift+P` → **Go: Install/Update Tools** → отметьте `dlv` и `dlv-dap`. В проекте включён legacy-адаптер Delve для Windows.

## Тесты

```bash
make test              # Go + web unit tests
make test-coverage     # с отчётом покрытия (100%)
make e2e-alpha         # API Flow A/B/C + QXLauncher dry-run + Playwright (Phase Alpha automated)
make e2e-dry-run       # API Flow A/B/C + QXLauncher launch dry-run (без JVM)
make build-launcher-win # bin/qx-launcher.exe (Windows)
```

CI (`.github/workflows/ci.yml`): unit tests, Playwright, `e2e-dry-run`, артефакт `qx-launcher-windows-amd64`.

## Структура репозитория

```text
services/
  qxapi/          QXApi — REST + WebSocket (Go)
  qxagent/        QXAgent — BYOS daemon (Go)
  qxlauncher/     QXLauncher — Windows (Go)
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

## Реализовано (MVP alpha — dev)

- [x] **Phase 0** — auth, profile, CI, 100% unit coverage
- [x] **Phase 1** — device link, instances, launch-bridge, QXLauncher + Vanilla
- [x] **Phase 2** — servers CRUD, SSH deploy, agent WSS; Stop/Restart/консоль при `minecraft_running`
- [x] **Phase 3** — registered user device status, JWT refresh в QXLauncher
- [x] **Phase Alpha (flows)** — manual Flow A/B/C ☑ ([test matrix](docs/qa/test-matrix.md), [FAQ](docs/faq.md))
- [ ] **Prod** — TLS, реальный VPS, smoke — **не готов** ([mvp §7.1](docs/mvp.md))

### Prod (Tier 0) — 🔲 заготовка, не validated

Скрипты и compose есть, но **к production пока не готовы** — см. [mvp §7.1](docs/mvp.md).

```bash
cp infra/docker/.env.prod.example infra/docker/.env.prod
# JWT, MySQL, SSH_MASTER_KEY — см. .env.prod.example
make prod-build
make prod-up
# → http://localhost:8080 (local smoke only; не prod-ready)
```
