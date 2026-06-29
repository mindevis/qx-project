# ADR-0011: Client Content — Local Install via QXLauncher

**Status:** Accepted  
**Date:** 2026-06-10

## Context

Mods, modpacks, shaders и resource packs — контент **игрового инстанса на ПК пользователя**.
Ранее в docs предполагался кэш файлов в MinIO с presigned URLs.

## Decision

- **Mods, modpacks, shaders, resource packs** устанавливаются **только на диск ПК** в папку инстанса через **QXLauncher**.
- **QXApi / MySQL** хранит **метаданные и manifest** (список файлов, URL, hashes) — **не бинарники**.
- QXLauncher скачивает напрямую с **CurseForge / Modrinth / Mojang** (authorized download URLs из API).
- **Локальный кэш** — на ПК (`QXLauncher` cache dir), не на MinIO.
- **MinIO** — только платформенные объекты: сборки QXLauncher, бэкапы выделенные серверов, skins (малые PNG), audit archive.

## Dedicated server

QXAgent устанавливает контент **на диск ноды** (прямые URL), без MinIO:

- **Modpack** — по manifest; состав путей зависит от `server_type`.
- **Mods** — только `forge` / `neoforge` / `fabric` / `quilt` / `hybrid` → `mods/`.
- **Plugins** — только `paper` / `spigot` / `purpur` / `hybrid` (Mohist, …) → `plugins/`.

См. [server-content-install.md](../server-content-install.md).

## Rationale

- Соответствует паттерну **Web-defined, Launcher-materialized**.
- Меньше legal-рисков redistribution через свой S3 ([security-legal.md §5](../security-legal.md)).
- Не нагружает dedicated server гигабайтами modpack-трафика.
- Как Prism / official launcher: файлы живут у пользователя.

## Consequences

- `modpacks-pipeline.md` — без шага «upload to MinIO»; server install — [server-content-install.md](../server-content-install.md).
- CF/MR rate limits — кэш metadata в Redis/MySQL; файлы — retry на клиенте.
- MVP может обойтись **без MinIO**, если нет auto-update builds и server backups.
