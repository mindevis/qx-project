# ADR-0003: Agent Linux-only, SSH Deploy

**Status:** Accepted
**Date:** 2026-06-09

## Context

BYOS серверы — преимущественно Linux VPS. Ручной pairing усложняет UX.

## Decision

- Agent **только Linux** (systemd).
- Установка: **backend SSH job** — upload binary, systemd unit, start.
- SSH private key хранится encrypted в MySQL.

## Rationale

- 99% MC hosting — Linux.
- SSH deploy из panel — один клик для админа.
- Windows agent отложен indefinitely.

## Consequences

- `internal/deploy/` в API с queue jobs.
- Agent pairing через JWT at deploy, не manual token paste.
- На VPS: `/etc/qx-agent/agent.toml` — см. [configuration.md](../configuration.md).
- Документация firewall: outbound 443 to QXApi.
