# ADR-0010: Own Launcher (Not GML Fork)

**Status:** Accepted
**Date:** 2026-06-09

## Decision

- **Do not fork GML.** Build **own QXLauncher** (Go tray) + **QXWeb** `/launcher` UI.
- GML/Aurora/TLauncher — **UX references only**, not codebases.

## Rationale

- Stack unified on Go backend/agent/tray.
- Full control over device linking, SSH deploy, launch bridge.
- Team size 2 — maintaining fork is costly.

## Consequences

- Implement modloader resolution in `internal/minecraft/` (reference Prism/MultiMC algorithms, not copy).
- Longer launcher phase vs fork — acceptable.
