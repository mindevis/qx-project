# QXSystem

Minecraft ecosystem: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent**, **QXMods**, **QXSkins**

**Prod platform ✅** ([mc.qx-dev.ru](https://mc.qx-dev.ru)) · docs [GitHub Pages](https://docs.qx-dev.ru)

[Contributing](CONTRIBUTING.md) · [Security](SECURITY.md) · [Code of Conduct](CODE_OF_CONDUCT.md) · [License](LICENSE)

## Требования

- Go 1.25+ (Go workspace)
- Node.js 24+
- Docker (MySQL, Redis, MinIO)
- [Go extension][go-ext] для отладки в Cursor / VS Code (Delve ставится автоматически)

[go-ext]: https://marketplace.visualstudio.com/items?itemName=golang.go

## Быстрый старт

```bash
cp qxapi.toml.example qxapi.toml
cp web.toml.example web.toml
cp launcher.toml.example launcher.toml
make jwt-secret-config   # или: make jwt-secret — только вывести в консоль
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
- Swagger UI: [localhost:3000/swagger/index.html](http://localhost:3000/swagger/index.html)
- Health: [health](http://localhost:3000/api/v1/health) · Ready: [health/ready](http://localhost:3000/api/v1/health/ready)

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
  qxagent/        QXAgent — dedicated server daemon (Go)
  qxlauncher/     QXLauncher — Windows (Go)
web/
  qxweb/          QXWeb — React SPA (panel + /launcher + /servers)
pkg/
  reporoot/       find repo root (go.work)
  mcmanifest/     Mojang manifest helpers
  protocol/       Agent ↔ API WSS types
qxapi.toml.example
web.toml.example
launcher.toml.example
agent.toml.example
infra/docker/     docker-compose dev stack
go.work           Go workspace
```
