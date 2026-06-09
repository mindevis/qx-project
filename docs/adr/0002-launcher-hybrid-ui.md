# ADR-0002: Hybrid Launcher (Go + React)

**Status:** Superseded by [ADR-0006](./0006-launcher-website-ui.md) (WebView removed)  
**Date:** 2026-06-09

## Context

Изначально рассматривался WebView (Wails) для React UI внутри desktop app.

## Decision (historical)

- ~~WebView~~ → UI на сайте `/launcher`
- **Go tray daemon** — JVM, sync, notifications
- **React on website** — данные через Backend REST API

## Rationale

- Переиспользование React/Antd между panel и launcher.
- Go — JVM spawn, Java download, updates без Electron bloat.
- Публичный список серверов — обычный API call из React UI.

## Consequences

- Два frontend apps: `web/panel-ui`, `web/launcher-ui`.
- CORS/auth: launcher UI uses same API; Go injects tokens into WebView storage on login.
- Local bridge: `POST http://127.0.0.1:<port>/launch` for game start only.
