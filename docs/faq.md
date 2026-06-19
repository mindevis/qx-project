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

### Почему кнопка «Играть» зависает на «ожидание tray»?

QXLauncher должен быть запущен и связан с тем же аккаунтом/guest-сессией. Проверьте иконку в системном трее и переменную `QX_API_BASE_URL`.

### Какие версии Minecraft поддерживаются?

MVP: **Vanilla only**, несколько версий через Mojang manifest (например 1.20.4, 1.21). Modloaders — v2+.

---

## Серверы

### Какие VPS подходят?

Linux x86_64 (Ubuntu 22.04+, Debian 12+), SSH по ключу. Подробнее: [ssh-deploy.md](./ssh-deploy.md).

### Deploy не подключает agent

- Проверьте SSH-ключ и firewall (исходящий 443 к API).
- В dev без бинарника агента включён dry-run (`QX_SSH_DEPLOY_DRY_RUN=1`).
- Укажите `QX_AGENT_BINARY_PATH` на сервере API для реального deploy.

---

## Prod / Self-Hosted

### Как поднять prod на одном VPS?

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
make e2e-dry-run    # API smoke + tray launch-bridge dry-run
make e2e-manual     # чеклист manual tray + JVM (Windows); -RunAll для e2e-alpha
```
