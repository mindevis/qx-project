# Production Deploy — Tier 0

Platform VPS: **Docker Compose** в `/opt/qxsystem`. Образы в **GHCR**, полностью автоматический деплой из **GitHub Actions**.

> **Домен:** `mc.qx-dev.ru` (панель + API `/api/v1`)  
> **VPS:** `178.172.136.26` · **Чеклист:** [mvp §7.1](./mvp.md#71-prod-readiness)

Push в `main` → сборка → GHCR → bootstrap VPS → `/opt/qxsystem` + `.env.prod` из Secrets → `docker compose up`.

**На VPS вручную ничего не копируете** — только DNS и секреты в GitHub.

---

## 1. Однократная настройка

### 1.1 DNS

A-запись `mc.qx-dev.ru` → IP platform VPS (`178.172.136.26`).

### 1.2 VPS

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
| `PROD_SSH_HOST` | IP VPS, напр. `178.172.136.26` |
| `PROD_SSH_USER` | SSH user (`root` или `ubuntu`) |
| `PROD_SSH_KEY` | Private key (OpenSSH PEM), полный ключ — **публичная часть** в `~/.ssh/authorized_keys` на VPS для `PROD_SSH_USER` |
| `GHCR_DEPLOY_TOKEN` | PAT с **`read:packages`** |
| `PROD_JWT_SECRET` | Подпись JWT — `make jwt-secret` (одна строка) |
| `PROD_SSH_MASTER_KEY` | `make ssh-master-key` (standard base64, 32 bytes) |
| `PROD_MYSQL_ROOT_PASSWORD` | Пароль root MySQL |
| `PROD_MYSQL_PASSWORD` | Пароль пользователя `qx` |
| `PROD_MINIO_PASSWORD` | Пароль MinIO |

Опционально в environment **protection rules**: Required reviewers — deploy на VPS только после approve.

**Environment variables** (не секреты, тот же environment `production`):

| Variable | Default |
| -------- | ------- |
| `VITE_API_BASE_URL` | `https://mc.qx-dev.ru/api/v1` |
| `CORS_ORIGIN` | `https://mc.qx-dev.ru` |
| `QX_PUBLIC_API_URL` | `https://mc.qx-dev.ru` |

> Можно положить те же ключи в **Repository secrets** — workflow подхватит их, если в environment нет значения. Для одного prod-VPS удобнее держать всё в **environment `production`**.

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

На VPS после deploy:

```
/opt/qxsystem/
  docker-compose.yml
  nginx/prod.conf
  schema.sql
  .env.prod           ← из GitHub Secrets (каждый deploy)
  image-tag.env       ← GHCR tags
  up.sh
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

## 3. TLS

После первого HTTP-smoke — Certbot на VPS (пока вручную):

```bash
sudo certbot certonly --standalone -d mc.qx-dev.ru
```

TLS в nginx — отдельный шаг (конфиг синхронизируется из репо при deploy).

---

## 4. Game VPS и QXLauncher

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
| `unable to authenticate` / `publickey` | Проверьте `PROD_SSH_HOST` / `PROD_SSH_USER`; на VPS в `~/.ssh/authorized_keys` должен быть **public** key, парный к `PROD_SSH_KEY`; ключ в Secret — целиком, с `-----BEGIN … KEY-----` |
| Bootstrap: sudo | Deploy-user нужен `sudo` или используйте `root` |
| `unauthorized` pull | `GHCR_DEPLOY_TOKEN` + read:packages |
| CORS | Variable `CORS_ORIGIN=https://mc.qx-dev.ru` |
| Смена секретов | Обновите Secret → re-run workflow |

---

## Связанные документы

[configuration.md](./configuration.md) · [ssh-deploy.md](./ssh-deploy.md) · [mvp §7.1](./mvp.md#71-prod-readiness)
