# ADR-0005: Modpack Sync via Shared modpack_id

**Status:** Accepted
**Date:** 2026-06-09

## Context

Игрок и админ должны играть на одной сборке: client instance + dedicated server.

## Decision

- Единый `modpack_id` на `launcher_instances` и `servers`.
- Один `QxModpackManifest` + `manifest_sha256` в MySQL.
- Client: QXLauncher install to PC.
- Server: QXAgent install to server disk; **mods vs plugins** by `server_type` ([post-mvp.md#server-content](../post-mvp.md#server-content)).

## Rationale

- Single source of truth в API.
- Hash mismatch → error, no desync gameplay.

## Consequences

- Panel UI: link modpack when creating instance and/or server.
- CurseForge/Modrinth import → normalize → store once.
