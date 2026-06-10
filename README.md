# QXProject

Minecraft ecosystem: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent** — каждый в своей папке.

Документация: [docs/architecture.md](docs/architecture.md) · [docs/mvp.md](docs/mvp.md)

## Требования

- Go 1.22+ (Go workspace)
- Node.js 20+
- Docker (MySQL, Redis, MinIO)
- [Go extension][go-ext] для отладки в Cursor / VS Code (Delve ставится автоматически)

[go-ext]: https://marketplace.visualstudio.com/items?itemName=golang.go

## Быстрый старт (Phase 0)

```bash
cp .env.example .env
make jwt-secret-env   # или: make jwt-secret — только вывести в консоль
make dev-up

# терминал 1 — QXApi
make tidy
make api

# терминал 2 — QXWeb
cd web/qxweb && npm install && npm run dev
```

- Web: [localhost:5173](http://localhost:5173)
- API base: [localhost:3000/api/v1](http://localhost:3000/api/v1)
- Health: [health](http://localhost:3000/api/v1/health) · Ready: [health/ready](http://localhost:3000/api/v1/health/ready)

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
```

CI (`.github/workflows/ci.yml`): `go test`, web `test:coverage`, build.

## Структура репозитория

```text
services/
  qxapi/          QXApi — REST + WebSocket (Go) — Phase 0 ✅
  qxagent/        QXAgent — BYOS daemon (Go, stub)
  qxlauncher/     QXLauncher — tray (Go, stub)
web/
  qxweb/          QXWeb — React SPA (panel + /launcher)
pkg/
  protocol/       Общие типы Agent ↔ API (WSS, placeholder)
infra/docker/     docker-compose dev stack
docs/             архитектура & ADR
go.work           Go workspace
```

Код сервиса **не смешивается**: у QXApi свой `internal/`, у QXAgent и QXLauncher — свои (по мере реализации).

## Phase 0 ✅

- [x] Monorepo по сервисам + Go workspace
- [x] Auth API (register, login, refresh, guest, logout, `/users/me`, change password/email)
- [x] QXWeb: landing, auth modal, profile (модалки email/пароль), `/launcher` (Phase 1 UI), placeholder `/servers`
- [x] Docker Compose dev (MySQL, Redis, MinIO)
- [x] CI + unit tests 100% (Go, React)

## Следующий шаг

**Phase 1** — device linking, instances CRUD, `/launcher`, QXLauncher tray (Windows, Vanilla). См. [docs/mvp.md](docs/mvp.md).
