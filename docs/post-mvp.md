# Запланировано (post-MVP)

Краткий список возможностей, которые **ещё не реализованы** или реализованы частично. Актуальный статус — в [architecture.md](./architecture.md) и [FAQ](./faq.md).

## Server content install {#server-content}

Установка mods/plugins/modpack **на game server** через QXAgent по `server_type` (Paper plugins, Forge mods и т.д.). См. заготовки в [api.md](./api.md), ADR [0011](./adr/0011-client-local-content-install.md).

## Modpacks pipeline {#modpacks}

Полная синхронизация modpack client ↔ server, CurseForge-first каталог на уровне modpack (QXMods покрывает поиск и установку отдельных модов на клиент). ADR: [0005](./adr/0005-modpack-sync.md), [0007](./adr/0007-curseforge-priority.md).

## Guest без регистрации {#guest}

`POST /auth/guest` и привязка лаунчера без аккаунта — v2+. Сейчас нужен JWT пользователя ([device-linking.md](./device-linking.md)).

## Auto-update лаунчера {#launcher-update}

Проверка версии и скачивание с сайта частично есть (`LAUNCHER_VERSION`, кнопка обновления). Полный канал релизов через MinIO manifest — в разработке. См. [configuration.md](./configuration.md), prod `.env` `LAUNCHER_DOWNLOAD_URL`.
