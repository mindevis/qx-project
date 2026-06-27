# ADR-0001: Tech Stack

**Status:** Accepted
**Date:** 2026-06-09

## Context

QXSystem — launcher + panel + agent. Команда: 1 senior + 1 junior. Нужен единый язык для backend/agent и зрелая
SPA-экосистема для UI.

## Decision

| Layer | Stack |
| ------- | ------- |
| QXWeb | TypeScript, React, Vite, Ant Design (SPA) |
| QXApi | Go, Gin, GORM |
| QXLauncher | Go |
| QXAgent | Go |
| DB | MySQL, Redis, MinIO |

## Rationale

- **Go** для API/agent/launcher shell — один язык, static binaries, хорош для WSS и process management.
- **React + Ant Design** — junior-friendly, богатые компоненты для panel и launcher UI.
- **GORM** — быстрый CRUD для маленькой команды.

## Consequences

- Go workspace: `services/qxapi`, `services/qxagent`, `services/qxlauncher` + `web/qxweb` (маршруты `/launcher/*`).
- Shared types в `pkg/protocol`.
- **Dev config:** TOML в корне репо — [configuration.md](../configuration.md); prod compose — `.env.prod`.
- Billing/post-MVP features не блокируют старт.
- Phase 0: Vitest (web) + `go test` (qxapi) — 100% unit coverage в CI; Phases 1–3 + MVP alpha (dev) ✅.
