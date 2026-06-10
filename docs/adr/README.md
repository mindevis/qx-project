# ADR Index

> Архитектурные решения зафиксированы до Phase 0. Реализация: [architecture.md §Статус реализации](../architecture.md).  
> REST API: единый префикс `/api/v1` (v1.6 docs).

| ADR | Title | Status |
| ----- | ------- | -------- |
| [0001](./0001-tech-stack.md) | Tech stack: Go backend, React frontend | Accepted |
| [0002](./0002-launcher-hybrid-ui.md) | Hybrid launcher (historical) | Superseded by 0006 |
| [0003](./0003-agent-linux-ssh-deploy.md) | Agent Linux-only, SSH deploy | Accepted |
| [0004](./0004-self-hosted-infra.md) | Self-hosted Docker Compose | Accepted |
| [0005](./0005-modpack-sync.md) | Shared modpack_id client ↔ server | Accepted |
| [0006](./0006-launcher-website-ui.md) | Launcher UI on website, no WebView | Accepted |
| [0007](./0007-curseforge-priority.md) | CurseForge primary modpack source | Accepted |
| [0008](./0008-launch-bridge-hybrid.md) | Hybrid launch: site POST + tray poll | Accepted |
| [0009](./0009-pure-self-hosted.md) | Pure self-hosted, no Cloudflare | Accepted |
| [0010](./0010-own-launcher-not-gml.md) | Own Go launcher, not GML fork | Accepted |
| [0011](./0011-client-local-content-install.md) | Client content on PC via QXLauncher, not MinIO | Accepted |
