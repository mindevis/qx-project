# Configuration — TOML

> Dev и локальные сервисы читают **TOML-файлы** в корне репозитория (или стандартные пути ОС).  
> **Не используйте** shell-переменные (`export`, `$env:`) для настройки QX.  
> **Prod Docker Compose** — исключение: `infra/docker/.env.prod` (стандарт compose).

---

## 1. Быстрый старт (dev)

```bash
cp qxapi.toml.example qxapi.toml
cp web.toml.example web.toml
cp launcher.toml.example launcher.toml
make jwt-secret-config   # записать jwt_secret в qxapi.toml
make dev-up
```

Локальные `*.toml` в `.gitignore` — не коммитьте секреты.

| Файл | Сервис | Шаблон |
| ------ | -------- | -------- |
| `qxapi.toml` | QXApi | `qxapi.toml.example` |
| `web.toml` | QXWeb (Vite) | `web.toml.example` |
| `launcher.toml` | QXLauncher | `launcher.toml.example` |
| `agent.toml` | QXAgent (local dev) | `agent.toml.example` |

---

## 2. QXApi — `qxapi.toml`

Путь: **корень репозитория** (рядом с `go.work`). Загрузка: `services/qxapi/cmd/run.go` → `internal/config.Load()`.

| Key | Default (dev) | Описание |
| ----- | ------------- | ---------- |
| `addr` | `:3000` | HTTP listen address |
| `gin_mode` | `debug` | Gin mode |
| `log_level` | `info` | `debug`, `info`, `warning`, `error` |
| `log_format` | `text` | `text` или `json` |
| `database_dsn` | MySQL localhost | GORM DSN |
| `jwt_secret` | dev placeholder | JWT signing; `make jwt-secret-config` |
| `access_token_ttl` | `15m` | JWT access TTL |
| `refresh_token_ttl` | `168h` | JWT refresh TTL |
| `cors_origin` | `http://localhost:5173` | CORS для QXWeb |
| `ssh_master_key` | dev default | base64, 32 bytes — шифрование SSH keys в DB |
| `public_api_url` | `http://localhost:3000` | URL API для agent.toml на VPS (Flow C) |
| `agent_binary_path` | auto-detect | Путь к `qx-agent-linux` для SSH deploy |

**Flow C (dev VPS):**

```toml
public_api_url = "http://host.docker.internal:3000"
```

Подсказки: `make dev-vps-info`.

---

## 3. QXWeb — `web.toml`

Путь: **корень репозитория**. Vite читает файл в `web/qxweb/vite.config.ts` и мапит ключи в `VITE_*`.

| Key | Default | Описание |
| ----- | --------- | ---------- |
| `api_base_url` | `http://localhost:3000/api/v1` | REST base для SPA |
| `log_level` | `debug` | Уровень логов фронтенда |
| `launcher_download_url` | — | URL кнопки «Скачать QXLauncher» |

---

## 4. QXLauncher — `launcher.toml`

**Dev:** корень репозитория (перекрывает `~/.qxlauncher/launcher.toml`, если оба существуют).  
**Установленный:** `%USERPROFILE%\.qxlauncher\launcher.toml` (Windows) или `~/.qxlauncher/launcher.toml`.

| Key | Default | Описание |
| ----- | --------- | ---------- |
| `api_base_url` | `http://localhost:3000/api/v1` | QXApi REST |
| `web_base_url` | `http://localhost:5173` | QXWeb для tray menu |
| `device_token_path` | `~/.qxlauncher/device_token` | Файл device JWT |
| `link_max_polls` | `60` | Poll device link |
| `skip_tray` | `false` | Консольный режим (без systray) |
| `launch_dry_run` | `false` | Launch-bridge без JVM |
| `device_id` | HWID ПК (UUID) | Стабильный ID машины; override для dev |
| `java_path` | — | Override Java binary |
| `skip_java_download` | `false` | Использовать system Java (тесты) |
| `email` / `password` | — | Опционально: auto-login на сайте (dev) |
| `log_level` / `log_format` | `info` / `text` | Логирование |

**HWID (`device_id`):** UUID v5 от идентификатора машины — Windows `MachineGuid`, Linux `/etc/machine-id`, macOS `IOPlatformUUID`. Кэш: `~/.qxlauncher/device_id`. Ссылка привязки: `{web_base_url}/launcher/link?device=<id>` — QXLauncher открывает её в браузере при первом запуске. Коды подтверждения не используются.

---

## 5. QXAgent — `agent.toml`

**Prod VPS:** `/etc/qxsystem/agent/agent.toml` (записывается при SSH deploy).  
**Local dev:** `agent.toml` в корне репо. Override: `qx-agent -config /path/to/agent.toml`.

| Key | Описание |
| ----- | ---------- |
| `agent_token` | JWT от deploy (обязателен) |
| `api_base_url` | REST base QXApi |
| `server_id` | UUID сервера |
| `server_root` | `/opt/qxsystem/server` |
| `ws_url` | Override WSS URL |
| `hostname` | Имя в heartbeat |
| `dry_run` | Не запускать JAR, только лог |
| `log_level` / `log_format` | Логирование |

---

## 6. Prod — `.env.prod` (GitHub Secrets)

Автогенерация при deploy: `infra/scripts/prod-render-env.sh`.  
Справочник ключей: `infra/docker/.env.prod.example`.  
**Гайд:** [production-deploy.md](./production-deploy.md).

| GitHub Secret (environment `production`) | Env в `.env.prod` |
| --------------- | ----------------- |
| `PROD_JWT_SECRET` | `JWT_SECRET` |
| `PROD_SSH_MASTER_KEY` | `SSH_MASTER_KEY` |
| `PROD_MYSQL_ROOT_PASSWORD` | `MYSQL_ROOT_PASSWORD` |
| `PROD_MYSQL_PASSWORD` | `MYSQL_PASSWORD`, `DATABASE_DSN` |
| `PROD_MINIO_PASSWORD` | `MINIO_ROOT_PASSWORD` |
| Variables `CORS_ORIGIN`, `QX_PUBLIC_API_URL` | одноимённые |

---

## 7. Приоритет и поиск файлов

```text
QXApi:     repo/qxapi.toml
QXWeb:     repo/web.toml → Vite env
Launcher:  ~/.qxlauncher/launcher.toml → repo/launcher.toml (repo wins in dev)
Agent:     -config flag → repo/agent.toml → /etc/qxsystem/agent/agent.toml
Repo root: pkg/reporoot (walk up to go.work)
```

---

*См. также: [README](../README.md), [production-deploy.md](./production-deploy.md), [architecture §10](./architecture.md), [services/README](../services/README.md)*

Последнее обновление: 2026-06-25 (prod single origin mc.qx-dev.ru)
