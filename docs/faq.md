# QXProject — FAQ (MVP Alpha)

Краткие ответы для закрытой beta. Полный scope: [mvp.md](./mvp.md).

---

## Лаунчер и игра

### Как начать играть без регистрации?

1. Скачайте и запустите **QXLauncher** (`make launcher` в dev).
2. Откройте ссылку `/launcher/link?device=…` из трея.
3. На сайте нажмите «Продолжить как гость».
4. На `/launcher` создайте Vanilla-инстанс → **Играть** (ник Player по умолчанию; свой ник — в offline-профиле).

### Как привязать лаунчер к аккаунту?

Войдите на сайте → откройте `/launcher/link?device=…` → «Связать устройство». Статус виден в профиле и на `/launcher`.

### Почему кнопка «Играть» зависает на «Запрос в очереди…»?

QXLauncher должен быть запущен и связан с тем же аккаунтом/guest-сессией. Проверьте иконку в системном трее и переменную `QX_API_BASE_URL`.

### Какие версии Minecraft поддерживаются?

MVP: **Vanilla only**, несколько версий через Mojang manifest (например 1.20.4, 1.21). Modloaders — v2+.

---

## Серверы

### Какие VPS подходят?

Linux x86_64 (Ubuntu 22.04+, Debian 12+), SSH по ключу. Подробнее: [ssh-deploy.md](./ssh-deploy.md).

### Deploy не подключает agent

- Проверьте SSH-ключ и firewall (исходящий HTTPS/HTTP к API).
- Deploy всегда выполняет SSH на VPS. Нужен бинарник агента (`make build-agent-linux` или `make dev-vps-up`).
- Укажите `QX_AGENT_BINARY_PATH` на сервере API для реального deploy.
- **Dev VPS:** в `.env` API — `QX_PUBLIC_API_URL=http://host.docker.internal:3000` (не `localhost` из контейнера).
- После **повторного Deploy** agent перезапускается через `systemctl restart` — ждите тег **Agent** в panel.

### Почему нет кнопки Start и консоли?

MVP UI показывает **Stop/Restart** и live-консоль только когда `minecraft_running === true`. Deploy agent ≠ запущенный Minecraft.

- Тег **Agent** (синий) — QXAgent подключён по WSS (`agent_online`).
- Статус MC — **offline**, пока JAR не запущен (вручную на VPS или через API `POST …/start`; кнопка Start в UI — post-MVP).

### Agent online, но Minecraft offline — это нормально?

Да. `agent_online` и `minecraft_running` — разные поля. После Deploy ожидайте Agent ☑ и MC offline до start JAR.

---

## Prod / Self-Hosted

> **Статус:** infra-скрипты есть, **к production пока не готовы**. Чеклист: [mvp §7.1](./mvp.md).

### Мы готовы к prod?

**Нет.** MVP alpha (Flow A/B/C в dev) пройден; prod требует TLS, реальный VPS, секреты и smoke — см. [test-matrix §8](./qa/test-matrix.md).

### Как поднять prod на одном VPS? *(когда будете готовы)*

```bash
cp infra/docker/.env.prod.example infra/docker/.env.prod
# отредактируйте секреты
make prod-up
```

Или: `bash infra/scripts/deploy.sh`. Стек: nginx + api + web + MySQL + Redis + MinIO. См. [architecture.md §9](./architecture.md).

### TLS и домены

MVP compose отдаёт HTTP на порту `HTTP_PORT` (8080). Для Let's Encrypt добавьте Certbot на хосте или расширьте nginx — [architecture.md §9.3](./architecture.md).

---

## Разработка

### Тесты

```bash
make test           # unit
make test-coverage  # 100% порог для qxapi и qxweb
make e2e-alpha        # API + dry-run + Playwright (всё автоматизированное)
make e2e-api-smoke  # API Flow A/B/C (router_test)
make e2e-dry-run    # API smoke + QXLauncher launch-bridge dry-run
make e2e-manual     # чеклист manual QXLauncher + JVM (Windows); -RunAll для e2e-alpha
```
