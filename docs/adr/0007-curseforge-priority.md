# ADR-0007: CurseForge Priority

**Status:** Accepted
**Date:** 2026-06-09

## Decision

- **CurseForge API key** — есть у команды, используется в production.
- **Приоритет каталога:** CurseForge **primary**, Modrinth secondary.
- Поиск modpacks: CF first; MR — если не найдено на CF или loader quilt/fabric-only on MR.

## Consequences

- `ModpackSvc` импорт: CF adapter первым в pipeline.
- UI: badge «CurseForge» default source filter.
- Cache warming: популярные CF modpacks в MinIO.
