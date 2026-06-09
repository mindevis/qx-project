# ADR-0004: Self-Hosted Infrastructure

**Status:** Accepted  
**Date:** 2026-06-09

## Decision

Production: **Docker Compose** on own VPS — Nginx, API, Web SPAs, PostgreSQL, Redis, MinIO.

## Rationale

- Минимальный budget ($5–30/mo).
- Полный контроль для команды из 2 человек.
- MinIO вместо S3; no managed cloud.

## Consequences

- Senior owns backups, TLS, deploy scripts.
- Tier 0 достаточен до thousands MAU.
