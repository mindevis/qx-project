# Launch Bridge — Site → QXLauncher → JVM

> **Статус реализации:** ✅ Phase 1 — API + QXLauncher + QXWeb «Играть»; manual JVM ☑ (I04, I05).  
> REST base: `/api/v1`.
> Решение **B1**: **гибрид** — QXWeb создаёт launch request в QXApi, QXLauncher poll'ит и запускает JVM.
> ADR: [0008](./adr/0008-launch-bridge-hybrid.md) · **Конфиг:** [configuration.md](./configuration.md) (`launcher.toml`)

---

## 1. Принцип

| Комponent | Роль |
| ----------- | ------ |
| **QXWeb `/launcher`** | UI: кнопка «Играть», выбор инстанса/аккаунта |
| **QXApi** | Очередь `launch_requests`, авторизация, manifest |
| **QXLauncher (tray)** | Poll, скачивание файлов, Mojang Java, spawn JVM |

QXLauncher **не имеет UI** — только tray icon, notifications, local engine.

---

## 2. Sequence

```mermaid
sequenceDiagram
    participant UI as QXWeb /launcher
    participant API as QXApi
    participant T as QXLauncher
    participant JVM as Minecraft

    Note over T: linked device_token, poll every 2s
    UI->>API: POST /api/v1/launcher/launch-requests { instance_id, profile_id }
    API->>API: Validate: device linked, user/guest RBAC
    API-->>UI: { request_id, status: queued }
    T->>API: GET /api/v1/launcher/launch-requests/pending
    API-->>T: { request_id, instance manifest, profile }
    T->>T: Ensure files, Java, classpath
    T->>API: PATCH /api/v1/launcher/launch-requests/{id} { status: running }
    T->>JVM: exec java ...
    T->>API: PATCH ... { status: completed | failed }
    UI->>API: GET /api/v1/launcher/launch-requests/{id} (poll optional)
    API-->>UI: running / completed / failed
```

---

## 3. API

### 3.1 Create (website)

```text
POST /api/v1/launcher/launch-requests
Authorization: Bearer <user_jwt> | Cookie guest + X-Device-Token
```

```json
{
  "instance_id": "uuid",
  "offline_profile_id": "uuid",
  "jvm_args_override": []
}
```

**RBAC:**

| Owner | Может launch |
| ------- | -------------- |
| Guest (linked) | Vanilla instances only |
| Registered user | All loaders + mods/shaders/resource packs |

Response `201`:

```json
{
  "id": "uuid",
  "status": "queued",
  "expires_at": "2026-06-09T12:05:00Z"
}
```

### 3.2 Poll pending (tray)

```text
GET /api/v1/launcher/launch-requests/pending
Authorization: Bearer <device_token>
```

Returns **at most one** oldest `queued` request for this device. Atomically marks `dispatched`.

### 3.3 Status update (tray)

```text
PATCH /api/v1/launcher/launch-requests/{id}
Authorization: Bearer <device_token>
```

```json
{ "status": "running", "pid": 12345 }
{ "status": "completed", "exit_code": 0 }
{ "status": "failed", "error": "JAVA_NOT_FOUND" }
```

### 3.4 UI poll (optional)

```text
GET /api/v1/launcher/launch-requests/{id}
```

For progress spinner on website.

---

## 4. State machine

```mermaid
stateDiagram-v2
    [*] --> queued: POST from website
    queued --> dispatched: tray GET pending
    dispatched --> running: JVM spawned
    running --> completed: process exit 0
    running --> failed: crash / error
    queued --> expired: TTL 5 min
    dispatched --> failed: tray timeout 60s
```

| Status | Meaning |
| -------- | --------- |
| `queued` | Waiting for tray |
| `dispatched` | Sent to tray, not started yet |
| `running` | JVM alive |
| `completed` | Clean exit |
| `failed` | Error |
| `expired` | Tray offline too long |

---

## 5. Tray loop

```go
// every 2 seconds when linked
for {
    req, err := api.FetchPendingLaunch(ctx, deviceToken)
    if req != nil {
        go executeLaunch(req) // download, java, spawn
    }
    sleep(2 * time.Second)
}
```

**Also poll:** instance sync (30s), launch requests (2s), device link status (3s when pending).

---

## 6. Local execution (tray)

1. Load instance manifest from API (or embedded in launch request).
2. Resolve Mojang Java — [mojang-java.md](./mojang-java.md).
3. Verify / download assets, libraries, mods.
4. Build classpath, natives, modloader processors.
5. `exec.Command(java, args...)` with log file `%APPDATA%/QX/logs/{instance_id}.log`.
6. Notify OS on failure; PATCH API status.

Dry-run (без JVM): `launch_dry_run = true` в `launcher.toml` — см. [configuration.md](./configuration.md).

---

## 7. Offline tray

If API unreachable but instance cached locally:

- **Guest:** block launch (must be online for first install).
- **Registered:** optional offline launch cached instance (post-MVP, feature flag).

MVP: **online required** for launch.

---

## 8. Security

| Risk | Mitigation |
| ------ | ------------ |
| Hijack launch request | Bound to `device_id`; only linked tray receives |
| Spam Play button | Rate limit 5/min per user — [security-legal.md](./security-legal.md) |
| Malicious manifest | Tray verifies SHA256 from API; API signs manifest server-side |

---

## 9. DDL

```sql
-- see schema.sql: launch_requests
```

---

*См. [device-linking.md](./device-linking.md), [api.md](./api.md), [mojang-java.md](./mojang-java.md), [configuration.md](./configuration.md)*

Последнее обновление: 2026-06-21 (Phase 1 ✅, HWID device link)
