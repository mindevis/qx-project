# ADR Index

> Архитектурные решения зафиксированы до Phase 0.  
> **Реализация:** Phase 0–3 + MVP alpha (dev) ✅ · Prod 🔲 — [architecture.md §Статус](../architecture.md), [mvp.md](../mvp.md).  
> REST API: единый префикс `/api/v1`.  
> **Конфиг (dev):** [configuration.md](../configuration.md) — TOML, не shell env.

| ADR | Title | Status |
| ----- | ------- | -------- |
| [0001](./0001-tech-stack.md) | Tech stack: Go backend, React frontend | Accepted · ✅ Phase 0 |
| [0002](./0002-launcher-hybrid-ui.md) | Hybrid launcher (historical) | Superseded by 0006 |
| [0003](./0003-agent-linux-ssh-deploy.md) | Agent Linux-only, SSH deploy | Accepted · ✅ Phase 2 |
| [0004](./0004-self-hosted-infra.md) | Self-hosted Docker Compose | Accepted · [production-deploy.md](../production-deploy.md) |
| [0005](./0005-modpack-sync.md) | Shared modpack_id client ↔ server | Accepted · post-MVP |
| [0006](./0006-launcher-website-ui.md) | Launcher UI on website, no WebView | Accepted · ✅ Phase 1 |
| [0007](./0007-curseforge-priority.md) | CurseForge primary modpack source | Accepted · post-MVP |
| [0008](./0008-launch-bridge-hybrid.md) | Hybrid launch: site POST + tray poll | Accepted · ✅ Phase 1 |
| [0009](./0009-pure-self-hosted.md) | Pure self-hosted, no Cloudflare | Accepted · [production-deploy.md §6](../production-deploy.md) |
| [0010](./0010-own-launcher-not-gml.md) | Own Go launcher, not GML fork | Accepted · ✅ Phase 1 |
| [0011](./0011-client-local-content-install.md) | Client content on PC via QXLauncher, not MinIO | Accepted · ✅ Vanilla MVP |

Последнее обновление: 2026-06-21
