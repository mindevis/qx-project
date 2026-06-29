# ADR-0009: Pure Self-Hosted (No Cloudflare)

**Status:** Accepted
**Date:** 2026-06-09

## Decision

- **No Cloudflare** proxy/CDN.
- TLS via Let's Encrypt on Nginx.
- DDoS: Nginx rate limits + fail2ban.
- **I8 dedicated server provider:** TBD at deploy time.

## Consequences

- Higher exposure to volumetric attacks — accept for MVP.
- MinIO + Nginx serve launcher releases, server backups, skins — **not** client/server modpack files ([ADR-0011](./0011-client-local-content-install.md)).
- Document egress IP for SSH deploy allowlists.

See [observability-ops.md](../observability-ops.md).
