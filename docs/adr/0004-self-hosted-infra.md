# ADR-0004: Self-Hosted Infrastructure

**Status:** Accepted
**Date:** 2026-06-09

## Decision

Production: **Docker Compose** on own VPS — Nginx, QXApi, QXWeb (React SPA), MySQL, Redis, MinIO (platform blobs only, [ADR-0011](./0011-client-local-content-install.md)).

## Rationale

- Минимальный budget ($5–30/mo).
- Полный контроль для команды из 2 человек.
- MinIO вместо S3; no managed cloud.

## Consequences

- Senior owns backups, TLS, deploy scripts.
- **Dev:** `*.toml` в корне репо — [configuration.md](../configuration.md).
- **Prod:** `infra/docker/.env.prod` + `docker-compose.prod.yml`.
- Tier 0 достаточен до thousands MAU.
