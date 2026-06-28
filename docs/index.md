# QXSystem

Minecraft ecosystem: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent**.

**Статус:** MVP alpha (dev) ✅ · Prod guide ☑ · Prod smoke 🔲

---

## Быстрые ссылки

| Документ | Описание |
| -------- | -------- |
| [FAQ](faq.md) | Частые вопросы — как начать играть, версии MC, серверы |
| [MVP](mvp.md) | Scope, Definition of Done, фазы |
| [Архитектура](architecture.md) | Полная архитектура и статус реализации |
| [API](api.md) | REST + WebSocket спецификация |
| [Production deploy](production-deploy.md) | Деплой на VPS (`mc.qx-dev.ru`) |
| [Конфигурация](configuration.md) | TOML-конфиг (dev) и prod `.env` |

## Компоненты

| Документ | Тема |
| -------- | ---- |
| [Device linking](device-linking.md) | Связь QXLauncher ↔ сайт |
| [Launch bridge](launch-bridge.md) | Сайт → QXLauncher → JVM |
| [Agent protocol](agent-protocol.md) | QXAgent WSS, deploy |
| [SSH deploy](ssh-deploy.md) | Установка агента на BYOS VPS |

## Репозиторий

Исходный код и issues: [github.com/mindevis/qx-project](https://github.com/mindevis/qx-project)

Локальный предпросмотр документации: `make docs-serve` (MkDocs Material).
