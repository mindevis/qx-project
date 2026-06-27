# QXSystem — FAQ (MVP Alpha)

Краткие ответы для закрытой beta (dev). **MVP alpha flows ✅ · Prod 🔲** — [mvp.md](./mvp.md).

---

## Лаунчер и игра

### Как начать играть без регистрации?

1. Скачайте и запустите **QXLauncher** (`make launcher` в dev).
2. Браузер **откроется сам** на странице привязки (HWID вашего ПК в URL).
3. Нажмите «Продолжить как гость».
4. На `/launcher` создайте Vanilla-инстанс → **Играть** (ник Player по умолчанию; свой ник — в offline-профиле).

### Как привязать лаунчер к аккаунту?

Запустите QXLauncher → браузер откроет `/launcher/link?device=…` → войдите на сайте (если ещё не вошли) → «Связать устройство». Статус виден в профиле и на `/launcher`.

### Почему кнопка «Играть» зависает на «Запрос в очереди…»?

QXLauncher должен быть запущен и связан с тем же аккаунтом/guest-сессией. Проверьте иконку в системном трее и `api_base_url` в `launcher.toml`.

### Какие версии Minecraft поддерживаются?

MVP: **Vanilla only**, несколько версий через Mojang manifest (например 1.20.4, 1.21). Modloaders — v2+.

---

## Серверы

### Какие VPS подходят?

Linux x86_64 (Ubuntu 22.04+, Debian 12+), SSH по ключу. Подробнее: [ssh-deploy.md](./ssh-deploy.md).

### Deploy не подключает agent

- Проверьте SSH-ключ и firewall (исходящий HTTPS/WSS к platform VPS).
- Deploy выполняет SSH на game VPS. Нужен бинарник: `make build-agent-linux` (prod: монтируется в API-контейнер).
- **Dev:** `agent_binary_path` в `qxapi.toml`; `public_api_url = "http://host.docker.internal:3000"`.
- **Prod:** `QX_PUBLIC_API_URL=https://api.qx-dev.ru`, `CORS_ORIGIN=https://mc.qx-dev.ru` — [production-deploy.md](./production-deploy.md).
- После **повторного Deploy** agent перезапускается через `systemctl restart`.

### Как управлять Minecraft-сервером?

1. **Servers** → VPS → **Add game server** (Vanilla/Paper/Forge/…).
2. Дождитесь статуса **running** или нажмите **Start**.
3. Откройте строку в таблице → страница сервера: RCON-консоль, настройки, моды, файлы.

### Почему нет консоли?

Консоль доступна на **странице игрового сервера**, когда статус `running`, `starting` или `installing` и агент онлайн. Deploy agent ≠ запущенный Minecraft.

---

## Prod / Self-Hosted

> **Гайд:** [production-deploy.md](./production-deploy.md) · Чеклист: [mvp §7.1](./mvp.md).

### Мы готовы к prod?

**Частично.** MVP alpha (Flow A/B/C в dev) пройден. Production требует выполнить чеклист §7.1: TLS, секреты, smoke на VPS, бэкапы.

### Как поднять prod на одном VPS?

См. **[production-deploy.md](./production-deploy.md)** — API `api.qx-dev.ru`, панель `mc.qx-dev.ru`, game VPS, QXLauncher.

```bash
cp infra/docker/.env.prod.qx-dev.example infra/docker/.env.prod
make jwt-secret && make build-agent-linux
make prod-build   # VITE_API_BASE_URL → api.qx-dev.ru
make prod-up
```

### TLS и домены

Prod: **два A-записи** (`api.qx-dev.ru`, `mc.qx-dev.ru`) → platform VPS. Compose отдаёт HTTP на `HTTP_PORT`; Certbot на оба имени — §6 в [production-deploy.md](./production-deploy.md).

---

## Разработка

### Конфигурация (dev)

TOML в корне репозитория — см. [configuration.md](./configuration.md):

```bash
cp qxapi.toml.example qxapi.toml
cp web.toml.example web.toml
cp launcher.toml.example launcher.toml
make jwt-secret-config
```

### Тесты

```bash
make test           # unit
make test-coverage  # 100% порог для qxapi и qxweb
make e2e-alpha        # API + dry-run + Playwright (всё автоматизированное)
make e2e-api-smoke  # API Flow A/B/C (router_test)
make e2e-dry-run    # API smoke + QXLauncher launch-bridge dry-run
make e2e-manual     # чеклист manual (все flows ☑ — см. test-matrix)
```

---

*См. [configuration.md](./configuration.md), [mvp.md](./mvp.md), [qa/test-matrix.md](./qa/test-matrix.md)*

Последнее обновление: 2026-06-21 (HWID + auto browser)
