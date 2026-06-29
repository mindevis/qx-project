# QXSystem

Minecraft ecosystem: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent**.

**Статус:** MVP alpha (dev) ✅ · **Prod platform ✅** ([mc.qx-dev.ru](https://mc.qx-dev.ru)) · docs [GitHub Pages](https://mindevis.github.io/qx-project/)

---

## Быстрые ссылки

| Документ | Описание |
| -------- | -------- |
| [FAQ](faq.md) | Частые вопросы — как начать играть, версии MC, серверы |
| [MVP](mvp.md) | Scope, Definition of Done, фазы |
| [Архитектура](architecture.md) | Полная архитектура и статус реализации |
| [API](api.md) | REST + WebSocket спецификация |
| [Production deploy](production-deploy.md) | **Prod live** — `mc.qx-dev.ru` (2026-06-29) |
| [Конфигурация](configuration.md) | TOML-конфиг (dev) и prod `.env` |

## Компоненты

| Документ | Тема |
| -------- | ---- |
| [Device linking](device-linking.md) | Связь QXLauncher ↔ сайт |
| [Launch bridge](launch-bridge.md) | Сайт → QXLauncher → JVM |
| [Agent protocol](agent-protocol.md) | QXAgent WSS, deploy |
| [SSH deploy](ssh-deploy.md) | Установка агента на dedicated server |

## Репозиторий

Исходный код и issues: [github.com/mindevis/qx-project](https://github.com/mindevis/qx-project)

Локальный предпросмотр документации: `make docs-serve` (MkDocs Material).
