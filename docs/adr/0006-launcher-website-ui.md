# ADR-0006: Launcher UI on Website (No WebView)

**Status:** Accepted (supersedes WebView part of ADR-0002)
**Date:** 2026-06-09

## Context

Изначально рассматривался WebView (Wails) для React UI внутри desktop app. Продуктовое решение: **внешний вид и
управление лаунчером — на сайте**.

## Decision

- **Go launcher** — tray daemon only: JVM, Mojang Java, sync, notifications, localhost bridge.
- **Launcher UI** — React SPA на сайте, маршрут `/launcher/*` (тот же `web/qxweb`).
- **Browser** — пользователь открывает сайт для инстансов; tray ЛКМ → `/launcher`.
- **WebView — не используется.**

## Rationale

- Один UI для panel и launcher (переиспользование Ant Design).
- Обновления UI без релиза desktop app.
- Проще для junior (только web).
- Device linking через браузер — естественный UX.

## Consequences

- Go app обязан: OS notifications, tray menu, poll link status.
- CORS: site and API same origin or configured.
- Game launch: site → `POST /api/v1/launcher/launch-requests` → QXLauncher poll ([launch-bridge.md](../launch-bridge.md), [ADR-0008](./0008-launch-bridge-hybrid.md)).

## Relation to ADR-0002

ADR-0002 «Hybrid» сохраняется частично: Go shell + React UI, но UI **hosted on website**, not embedded.
