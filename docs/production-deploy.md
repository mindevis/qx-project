# Production Deploy — Tier 0

Platform host: **Docker Compose** в `/opt/qxsystem`. Образы в **GHCR**, полностью автоматический деплой из **GitHub Actions**.

> **Домен:** `mc.qx-dev.ru` (панель + API `/api/v1`)  
> **Platform host:** `178.172.136.26` · **Чеклист:** [mvp §7.1](./mvp.md#71-prod-readiness)  
> **Статус:** ✅ **production live** (2026-06-29)

Push в `main` → **CI green** → **Prod release** → GHCR → bootstrap platform host → `/opt/qxsystem` + `.env.prod` из Secrets → `docker compose up`.

**На dedicated server вручную ничего не копируете** — только DNS и секреты в GitHub.

---

## 0. Статус production

| Проверка | Результат |
| -------- | --------- |
| Workflow [prod-release.yml](https://github.com/mindevis/qx-project/blob/main/.github/workflows/prod-release.yml) | ✅ deploy после успешного CI |
| Панель | ✅ [http://mc.qx-dev.ru](http://mc.qx-dev.ru) |
| API health | ✅ `GET /api/v1/health` → `{"status":"ok"}` |
| Стек на dedicated server | QXApi, QXWeb, MySQL, Redis, MinIO, Nginx в `/opt/qxsystem` |
| TLS (HTTPS) | ☑ автоматически при `PROD_CLOUDFLARE_API_TOKEN` + `PROD_CERTBOT_EMAIL` — [§3 TLS](#3-tls) |

---

## 1. Однократная настройка

### 1.1 DNS

A-запись `mc.qx-dev.ru` → IP Platform host (`178.172.136.26`).

### 1.2 Подготовка хоста

Свежий Ubuntu/Debian с **SSH-доступом** для deploy-пользователя (рекомендуется `root` или user с `sudo` без пароля).

При **первом** deploy workflow сам:

- установит Docker (если нет);
- создаст `/opt/qxsystem/nginx` и выставит права;
- запишет `.env.prod` из Secrets;
- подтянет образы и запустит stack.

### 1.3 GitHub Secrets — Environment `production` (рекомендуется)

**Repository secrets** — проще, но доступны всем workflow в репо.  
**Environment secrets** — лучше для prod: секреты только у job deploy, можно включить **Required reviewers** перед выкладкой.

Создайте environment: **Settings → Environments → New environment → `production`**

Добавьте **Environment secrets** (имена те же):

| Secret | Описание |
| ------ | -------- |
| `PROD_SSH_HOST` | IP platform host, напр. `178.172.136.26` |
| `PROD_SSH_USER` | SSH user (`root` или `ubuntu`) |
| `PROD_SSH_KEY` | Private key (OpenSSH PEM), полный ключ — **публичная часть** в `~/.ssh/authorized_keys` на dedicated server для `PROD_SSH_USER` |
| `GHCR_DEPLOY_TOKEN` | PAT с **`read:packages`** |
| `PROD_JWT_SECRET` | Подпись JWT — `make jwt-secret` (одна строка) |
| `PROD_SSH_MASTER_KEY` | `make ssh-master-key` (standard base64, 32 bytes) |
| `PROD_MYSQL_ROOT_PASSWORD` | Пароль root MySQL |
| `PROD_MYSQL_PASSWORD` | Пароль пользователя `qx` |
| `PROD_MINIO_PASSWORD` | Пароль MinIO |
| `CURSEFORGE_API_KEY` | *(опционально)* API key из [CurseForge for Studios](https://console.curseforge.com/) → **API Keys** (часто начинается с `$2a$10$` — это нормально). После добавления/смены — **Actions → Prod release → Run workflow** |
| `PROD_MOJANG_CLIENT_SECRET` | *(опционально)* Secret Value Azure AD app для Mojang OAuth |
| `QX_PUBLIC_API_URL` | *(опционально)* публичный URL API для agent deploy |
| `VITE_API_BASE_URL` | *(опционально)* base URL в **сборке** QXWeb; по умолчанию `/api/v1` (same-origin) |
| `PROD_CLOUDFLARE_API_TOKEN` | *(опционально)* Cloudflare API token с **Zone → DNS → Edit** для зоны `qx-dev.ru` — выпуск LE через DNS-01 |
| `PROD_CERTBOT_EMAIL` | *(опционально)* email для Let's Encrypt (обязателен вместе с Cloudflare token) |

Опционально в environment **protection rules**: Required reviewers — deploy на dedicated server только после approve.

**Environment variables** (альтернатива secrets для несекретных URL):

| Variable | Default |
| -------- | ------- |
| `VITE_API_BASE_URL` | `/api/v1` (относительный — работает по HTTP до TLS) |
| `CORS_ORIGIN` | `https://mc.qx-dev.ru` |
| `QX_PUBLIC_API_URL` | `https://mc.qx-dev.ru` |

> `CORS_ORIGIN`, `QX_PUBLIC_API_URL`, `VITE_API_BASE_URL` можно держать в **Secrets** или **Variables** — workflow читает оба. Секреты `PROD_*` обязательны.

PAT для `GHCR_DEPLOY_TOKEN`: Developer settings → Fine-grained token → **read packages** для репозитория.

#### Как сгенерировать `PROD_JWT_SECRET` и `PROD_SSH_MASTER_KEY`

Это **два разных секрета**. Одной командой оба не выдаются.

```bash
# 1) JWT — подпись access/refresh токенов
make jwt-secret
# → GitHub Secret PROD_JWT_SECRET

# 2) SSH master key — шифрование SSH private keys в MySQL
make ssh-master-key
# → GitHub Secret PROD_SSH_MASTER_KEY

# или оба сразу:
make prod-secrets
```

Работает на Windows без OpenSSL (нужен только Go).

---

## 2. Деплой

Workflow: [`.github/workflows/prod-release.yml`](https://github.com/mindevis/qx-project/blob/main/.github/workflows/prod-release.yml)

| Триггер | Действие |
| ------- | -------- |
| Push в `main` после **успешного CI** | build → GHCR → deploy (только если менялись `services/`, `web/`, `infra/docker/`, `pkg/` и т.п.) |
| Actions → **Prod release** → Run | то же (ручной запуск, без ожидания CI) |

> **Смена секретов** (например `CURSEFORGE_API_KEY`, `PROD_MOJANG_CLIENT_SECRET`) не попадает в path filter — после добавления ключа в GitHub обязательно запустите **Prod release** вручную, иначе `.env.prod` на сервере не обновится.

**Проверка CurseForge на сервере** (ключ из консоли часто выглядит как `$2a$10$…` — это нормально; в `.env.prod` каждый `$` должен быть записан как `$$`):

```bash
grep CURSEFORGE_API_KEY /opt/qxsystem/.env.prod   # в файле $$ = один $ в контейнере
docker compose exec api printenv CURSEFORGE_API_KEY | wc -c   # ~61 для типичного ключа (+ перевод строки)
```

Если `docker compose` пишет `variable is not set` для фрагмента ключа — в `.env.prod` неэкранированные `$`; `prod-render-env.sh` экранирует их при deploy. Ручной hotfix:

```bash
# в /opt/qxsystem/.env.prod — удвоить каждый $:
# CURSEFORGE_API_KEY=$$2a$$10$$DGBd1hfp5E98sgPzS8xMBOTEISfEZG451UGeeemxA7F3ogg2CaAgi
cd /opt/qxsystem && ./merge-compose-env.sh /opt/qxsystem && docker compose up -d api
```

На dedicated server после deploy:

```
/opt/qxsystem/
  docker-compose.yml
  nginx/prod-http.conf
  nginx/prod-tls.conf
  nginx/active.conf   ← выбирается up.sh (HTTP или TLS)
  certbot-cloudflare.sh
  schema.sql
  .env.prod           ← из GitHub Secrets (каждый deploy)
  image-tag.env       ← GHCR tags
  .env                ← merge .env.prod + image-tag.env (для `docker compose` без флагов)
  up.sh
```

**На dedicated server:**

```bash
cd /opt/qxsystem
./up.sh                    # pull + up (как в CI)
docker compose ps          # после up.sh — читает .env
curl -s http://127.0.0.1/api/v1/health
```

Образы: `ghcr.io/<owner>/qx-api:prod-<sha>`, `ghcr.io/<owner>/qx-web:prod-<sha>` (+ тег `prod`).

Nginx маршрутизация на `mc.qx-dev.ru`:

| Path | Backend |
| ---- | ------- |
| `/api/*` | QXApi |
| `/agent/*` | QXApi (WSS) |
| `/swagger/*` | QXApi |
| `/` | QXWeb SPA |

---

## 3. TLS (HTTPS через Cloudflare + Let's Encrypt)

При каждом deploy `up.sh` автоматически выпускает сертификат, если в GitHub Environment заданы **`PROD_CLOUDFLARE_API_TOKEN`** и **`PROD_CERTBOT_EMAIL`**. Используется **DNS-01** через Cloudflare — порт 80 на dedicated server для выпуска не нужен.

### 3.1 Cloudflare

1. **DNS:** A-запись `mc.qx-dev.ru` → IP platform host (оранжевое облако — OK).
2. **SSL/TLS → Overview:** режим **Full (strict)** (origin с валидным LE-сертификатом).
3. **API Token** ([My Profile → API Tokens](https://dash.cloudflare.com/profile/api-tokens)):
   - шаблон *Edit zone DNS* или custom token;
   - permissions: **Zone → DNS → Edit** для зоны `qx-dev.ru`;
   - сохраните в GitHub Secret **`PROD_CLOUDFLARE_API_TOKEN`**.

### 3.2 GitHub Secrets

| Secret | Пример |
| ------ | ------ |
| `PROD_CERTBOT_EMAIL` | `admin@example.com` |
| `PROD_CLOUDFLARE_API_TOKEN` | токен из §3.1 |

Опционально в Variables: `CORS_ORIGIN=https://mc.qx-dev.ru`, `QX_PUBLIC_API_URL=https://mc.qx-dev.ru`, `VITE_API_BASE_URL=https://mc.qx-dev.ru/api/v1` (или оставьте `/api/v1`).

### 3.3 Деплой

Re-run workflow **Prod release** (или push в `main`). На dedicated server:

```bash
cd /opt/qxsystem
./up.sh
curl -s https://mc.qx-dev.ru/api/v1/health
```

Сертификат: `/etc/letsencrypt/live/mc.qx-dev.ru/`. Продление — системный `certbot.timer`; после renew nginx перезагружается hook'ом.

Без Cloudflare token стек остаётся на HTTP (`nginx/prod-http.conf`).

### 3.4 Firewall

Откройте **443/tcp** на dedicated server (и 80 для редиректа HTTP→HTTPS).

---

## 4. Game dedicated server и QXLauncher

> **Hostname игрового сервера** не должен совпадать с `QX_PUBLIC_API_URL` (`mc.qx-dev.ru`). Рекомендуется `mcs.qx-dev.ru`; `api_base_url` в agent.toml остаётся `https://mc.qx-dev.ru/api/v1`.

1. **Servers** → Deploy agent ([ssh-deploy.md](./ssh-deploy.md))
2. **Add game server** → Start
3. Лаунчер:

```toml
api_base_url = "https://mc.qx-dev.ru/api/v1"
web_base_url = "https://mc.qx-dev.ru"
```

---

## 5. Troubleshooting

| Проблема | Решение |
| -------- | ------- |
| `Missing secret` | Заполните все Secrets из §1.3 |
| `QX_API_IMAGE is missing` / `MYSQL_PASSWORD` not set | Не запускайте голый `docker compose` — только `./up.sh` или после него `docker compose ps` (нужен файл `.env`) |
| `unable to authenticate` / `publickey` | Проверьте `PROD_SSH_HOST` / `PROD_SSH_USER`; на dedicated server в `~/.ssh/authorized_keys` должен быть **public** key, парный к `PROD_SSH_KEY`; ключ в Secret — целиком, с `-----BEGIN … KEY-----` |
| **«Сервер недоступен»** в UI | API не отвечает: `curl http://127.0.0.1/api/v1/health` на dedicated server. Если локально OK, а в браузере нет — QXWeb собран с `https://…` без TLS: пересоберите с `VITE_API_BASE_URL=/api/v1` или включите [§3 TLS](#3-tls) |
| Bootstrap: sudo | Deploy-user нужен `sudo` или используйте `root` |
| `unauthorized` pull | `GHCR_DEPLOY_TOKEN` + read:packages |
| CORS | `CORS_ORIGIN` = origin сайта (`https://mc.qx-dev.ru` после TLS) |
| HTTPS не открывается | Проверьте `443` на firewall; Cloudflare SSL = **Full (strict)**; secrets `PROD_CLOUDFLARE_API_TOKEN` + `PROD_CERTBOT_EMAIL`; логи: `sudo certbot certificates` на dedicated server |
| Agent `dial tcp 127.0.1.1:443: connection refused` | **Co-located** (API и agent на одном хосте): `./up.sh`, затем `api_base_url = "http://127.0.0.1:3000/api/v1"` в `/etc/qxsystem/agent/agent.toml`, `sudo systemctl restart qx-agent`. **Remote** game server: hostname **не должен** совпадать с `QX_PUBLIC_API_URL` — переименуйте в `mcs.qx-dev.ru` (`hostnamectl`, `/etc/hosts`); `api_base_url` остаётся `https://mc.qx-dev.ru/api/v1`. Проверка: `getent hosts mc.qx-dev.ru` → публичный IP платформы, не `127.0.1.1` |
| Смена секретов | Обновите Secret → re-run workflow |

---

## Связанные документы

[configuration.md](./configuration.md) · [ssh-deploy.md](./ssh-deploy.md) · [mvp §7.1](./mvp.md#71-prod-readiness)

Последнее обновление: 2026-06-29 (prod platform live)
