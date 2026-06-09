# ADR-0008: Hybrid Launch Bridge

**Status:** Accepted
**Date:** 2026-06-09

## Decision

Website `POST /launcher/launch-requests` + Go tray polls `GET .../pending` + local JVM spawn.

Not pure localhost (browser CORS issues). Not pure push WebSocket (complexity).

## Rationale

- Website has no direct access to tray process.
- API mediates with device_token binding.
- Works when browser and tray on same PC without open ports.

## Consequences

- `launch_requests` table + TTL cleanup job.
- Tray poll interval 2s when linked.
- UI optional poll for status spinner.

See [launch-bridge.md](../launch-bridge.md).
