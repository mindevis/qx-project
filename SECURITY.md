# Security Policy

## Supported versions

| Version / branch | Supported |
| ---------------- | --------- |
| `main` (prod at [mc.qx-dev.ru](https://mc.qx-dev.ru)) | ✅ |
| Older commits / forks | ❌ |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

1. Email **mindevis.by@gmail.com** with:
   - описание уязвимости и шаги воспроизведения;
   - затронутые компоненты (`qxapi`, `qxagent`, `qxlauncher`, `qxweb`, infra);
   - оценка impact (RCE, auth bypass, data leak и т.д.);
   - при возможности — PoC или patch suggestion.
2. Либо используйте [GitHub Private Security Advisory](https://github.com/mindevis/qx-project/security/advisories/new) для репозитория.

Ожидайте подтверждение в течение **72 часов**. Мы согласуем срок исправления и публикацию advisory после патча.

## In scope

- QXApi (REST, WebSocket agent hub, auth)
- QXAgent (WSS, server process control)
- QXLauncher (device link, launch bridge)
- QXWeb (панель, API client)
- Production deploy (`/opt/qxsystem`, nginx, secrets handling)

## Out of scope

- Уязвимости сторонних Minecraft-серверов (Paper, Forge и т.д.) на game dedicated server
- Социальная инженерия, физический доступ к dedicated server
- DoS без демонстрации разумного impact на QX stack

## Secure development

- Секреты только в GitHub Environment / dedicated server `.env.prod`, не в репозитории.
- Dependabot и CodeQL включены — см. [Security](https://github.com/mindevis/qx-project/security).
- Подробнее: [docs/security-legal.md](docs/security-legal.md).

## Recognition

Имена репортёров публикуются только с явного согласия.
