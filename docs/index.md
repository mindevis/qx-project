# QXSystem

Экосистема Minecraft: **QXWeb**, **QXApi**, **QXLauncher**, **QXAgent**.

**Prod:** [mc.qx-dev.ru](https://mc.qx-dev.ru) · **Документация:** этот сайт · **Код:** [github.com/mindevis/qx-project](https://github.com/mindevis/qx-project)

---

## С чего начать

| Документ | Для кого |
| -------- | -------- |
| [FAQ](faq.md) | Игроки и админы — как играть, привязать лаунчер |
| [Конфигурация](configuration.md) | Dev — TOML-файлы в корне репо |
| [Production deploy](production-deploy.md) | Ops — деплой на `mc.qx-dev.ru` |
| [API](api.md) | Разработчики — REST и WebSocket |

## Лаунчер и игра

| Документ | Тема |
| -------- | ---- |
| [Device linking](device-linking.md) | Связь QXLauncher ↔ сайт |
| [Launch bridge](launch-bridge.md) | Сайт → очередь → JVM |
| [QX Cosmetics](cosmetics.md) | Скины без модов на клиенте |

## Инфраструктура

| Документ | Тема |
| -------- | ---- |
| [Agent protocol](agent-protocol.md) | QXAgent, deploy, WSS |
| [SSH deploy](ssh-deploy.md) | Установка агента на game VPS |
| [Архитектура](architecture.md) | Полное описание системы |
| [Безопасность](security-legal.md) | Секреты, Mojang, CurseForge |

Локальный предпросмотр: `make docs-serve` · сборка как в CI: `make docs-build`.
