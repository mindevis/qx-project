# ADR-0001: Tech Stack

**Status:** Accepted
**Date:** 2026-06-09

## Context

QXProject — launcher + panel + agent. Команда: 1 senior + 1 junior. Нужен единый язык для backend/agent и зрелая
SPA-экосистема для UI.

## Decision

| Layer | Stack |
| ------- | ------- |
| Panel Web | TypeScript, React, Vite, Ant Design (SPA) |
| Backend API | Go, Gin, GORM |
| Launcher native | Go |
| Agent | Go |
| DB | PostgreSQL, Redis, MinIO |

## Rationale

- **Go** для API/agent/launcher shell — один язык, static binaries, хорош для WSS и process management.
- **React + Ant Design** — junior-friendly, богатые компоненты для panel и launcher UI.
- **GORM** — быстрый CRUD для маленькой команды.

## Consequences

- Monorepo Go modules + `web/panel-ui` + `web/launcher-ui`.
- Shared types в `pkg/protocol`.
- Billing/post-MVP features не блокируют старт.
