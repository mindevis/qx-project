Краткие ответы для **prod** ([mc.qx-dev.ru](https://mc.qx-dev.ru)) и dev. **MVP alpha flows ✅ · Prod platform ✅** — [mvp.md](./mvp.md).

---

## Production

### Где открыть панель?

**[https://mc.qx-dev.ru](https://mc.qx-dev.ru)** (или `http://` до настройки TLS). API: `https://mc.qx-dev.ru/api/v1`.

### Как устроен deploy?

Push в `main` → CI → автоматический **Prod release** → образы в GHCR → VPS `/opt/qxsystem`. Подробно: [production-deploy.md](./production-deploy.md).

---

## Лаунчер и игра

### Как начать играть?

1. **Зарегистрируйтесь или войдите** на сайте (email + пароль).
2. Скачайте и запустите **QXLauncher** (`make launcher` в dev) — браузер откроет `/launcher/link?device=…`.
3. Нажмите **«Связать устройство»** (нужна активная сессия на сайте).
4. На `/launcher` создайте инстанс → offline-профиль (ник) → **Играть**.

QXLauncher должен оставаться запущенным (иконка в трее). Без привязки устройства инстансы и запуск недоступны.

### Можно ли без регистрации?

**В текущей версии — нет.** Раньше планировался guest-flow («Продолжить как гость»), но в UI и API он отключён: `POST /auth/guest` не зарегистрирован в роутере, привязка требует JWT пользователя. Guest и merge guest→user — **v2+** ([device-linking.md](./device-linking.md) §5, [mvp.md](./mvp.md)).

### Почему кнопка «Играть» зависает на «Запрос в очереди…»?

QXLauncher должен быть запущен и привязан к **тому же аккаунту**, под которым вы нажали «Играть». Проверьте иконку в системном трее и `api_base_url` в `launcher.toml`.

### Какие версии Minecraft поддерживаются?

**Клиент (QXLauncher):** Vanilla, **Forge**, **NeoForge**, **Fabric** и **Quilt**. Список версий Minecraft подтягивается из Mojang manifest (актуальные релизы, например 1.20.4, 1.21.x). Для modloader'ов дополнительно выбирается версия загрузчика — доступные пары MC + loader показываются при создании инстанса на `/launcher`.

**Игровые серверы на VPS:** Vanilla; плагиновые (Paper, Spigot, Purpur); модовые (Forge, NeoForge, Fabric, Quilt); гибридные (Mohist, Magma, Arclight). Версии MC — из тех же источников, что и при создании сервера в панели.

Пока **вне scope:** автоматические modpack'и (CurseForge/Modrinth), загрузка шейдеров и resource pack'ов из панели — v2+.

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
make e2e-api-smoke  # API Flow A/C (router_test)
make e2e-dry-run    # API smoke + QXLauncher launch-bridge dry-run
make e2e-manual     # чеклист manual (все flows ☑ — см. test-matrix)
```

### Документация (GitHub Pages)

Сайт: **[mindevis.github.io/qx-project](https://mindevis.github.io/qx-project/)** — собирается MkDocs Material при push в `main` (workflow `.github/workflows/docs.yml`).

Локально: `make docs-serve` (нужен Python 3.12+).

Первый раз в репозитории: **Settings → Pages → Build and deployment → Source: GitHub Actions**.

---

*См. [configuration.md](./configuration.md), [mvp.md](./mvp.md), [qa/test-matrix.md](./qa/test-matrix.md)*

Последнее обновление: 2026-06-29 (prod platform live)
