# Contributing to QXSystem

Спасибо за интерес к проекту! QXSystem — экосистема Minecraft: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent**.

Документация: [mindevis.github.io/qx-project](https://mindevis.github.io/qx-project/)

## Перед началом

1. Прочитайте [README.md](README.md) и [docs/mvp.md](docs/mvp.md) — scope MVP и границы v1.
2. Для багов и идей используйте [Issues](https://github.com/mindevis/qx-project/issues) (шаблоны в `.github/ISSUE_TEMPLATE/`).
3. Соблюдайте [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Dev-окружение

**Требования:** Go 1.25+, Node.js 24+, Docker.

```bash
cp qxapi.toml.example qxapi.toml
cp web.toml.example web.toml
cp launcher.toml.example launcher.toml
make jwt-secret-config
make dev-up
make api          # терминал 1
cd web/qxweb && npm install && npm run dev   # терминал 2
```

Подробнее: [docs/configuration.md](docs/configuration.md).

## Pull requests

1. Форк / ветка от `main`.
2. Небольшие, сфокусированные изменения — один PR на одну задачу.
3. Перед push локально:

```bash
# Go
cd services/qxapi && go test ./...
cd services/qxagent && go test ./...
cd services/qxlauncher && go test ./...

# Web
cd web/qxweb && npm ci && npm run lint && npm run test:coverage && npm run build
```

4. CI должен быть зелёным (`.github/workflows/ci.yml`).
5. Обновите документацию в `docs/`, если меняется поведение API, deploy или UX.
6. Не коммитьте секреты (`.env`, ключи, `*.toml` с паролями).

Шаблон PR: [.github/pull_request_template.md](.github/pull_request_template.md).

## Стиль кода

- **Go:** `go vet`, существующие паттерны в `services/` и `pkg/`.
- **TypeScript/React:** ESLint (`npm run lint`), тесты Vitest + Playwright e2e для UI.
- Минимальный diff — без рефакторинга «заодно».

## Безопасность

Уязвимости сообщайте приватно — см. [SECURITY.md](SECURITY.md). Не открывайте public issue с exploit-деталями.

## Prod deploy

Изменения в `infra/`, `services/`, `web/` на `main` после CI могут триггерить [prod release](docs/production-deploy.md). Будьте осторожны с breaking changes в API и миграциями БД (`docs/schema.sql`).
