# ADR-0005: Modpack Sync via Shared modpack_id

**Status:** Accepted
**Date:** 2026-06-09

## Context

Игрок и админ должны играть на одной сборке: client instance + BYOS server.

## Decision

- Единый `modpack_id` на `launcher_instances` и `servers`.
- Один `QxModpackManifest` + `manifest_sha256` в MySQL.
- Client: Go launcher install.
- Server: agent `cmd.modpack.install` with hash verification.

## Rationale

- Single source of truth в API.
- Hash mismatch → error, no desync gameplay.

## Consequences

- Panel UI: link modpack when creating instance and/or server.
- CurseForge/Modrinth import → normalize → store once.
