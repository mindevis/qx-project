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
- Deploy выполняет SSH на game VPS. Бинарник QXAgent встроен в API-образ при `make prod-pack`.
- **Dev:** `agent_binary_path` в `qxapi.toml`; `public_api_url = "http://host.docker.internal:3000"`.
- **Prod:** `QX_PUBLIC_API_URL=https://mc.qx-dev.ru`, `CORS_ORIGIN=https://mc.qx-dev.ru` — [production-deploy.md](./production-deploy.md).
- После **повторного Deploy** agent перезапускается через `systemctl restart`.

### Как управлять Minecraft-сервером?

1. **Servers** → VPS → **Add game server** (Vanilla/Paper/Forge/…).
2. Дождитесь статуса **running** или нажмите **Start**.
3. Откройте строку в таблице → страница сервера: RCON-консоль, настройки, моды, файлы.

### Почему нет консоли?

Консоль доступна на **странице игрового сервера**, когда статус `running`, `starting` или `installing` и агент онлайн. Deploy agent ≠ запущенный Minecraft.

---

## Prod / Self-Hosted

Push в `main` → полный автодеплoy (Docker, `/opt/qxsystem`, `.env.prod` из GitHub Secrets).  
**[production-deploy.md](./production-deploy.md)** — таблица Secrets.

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

Последнее обновление: 2026-06-25 (docs cleanup)
