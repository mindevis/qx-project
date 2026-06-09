# QXAgent Protocol

> Версия: **1.0** · Transport: **WebSocket (WSS)** · Format: **JSON**
> Shared types: `pkg/protocol` (Go)

---

## 1. Обзор

QXAgent — Go daemon на **Linux**, systemd service. QXApi разворачивает QXAgent через **SSH** (см. §2). После deploy
QXAgent устанавливает **WSS** к Agent Hub и обменивается командами/events.

| Свойство | Значение |
| ---------- | ---------- |
| Platform | **Linux only** (amd64, arm64 TBD) |
| Transport | WSS (`wss://api.qx.example.com/agent/v1/connect`) |
| Auth | JWT agent token (issued at deploy) |
| Idempotency | `request_id` UUID на каждую command |
| Ordering | At-least-once delivery; agent dedupe by `request_id` |

---

## 2. SSH Deploy (установка)

Backend выполняет deploy job — agent **не** ставится вручную пользователем.

### 2.1 Flow

```mermaid
sequenceDiagram
    participant U as Admin (Panel)
    participant API as Backend
    participant VPS as Linux VPS
    participant A as QXAgent

    U->>API: POST /servers {ssh, server_type, modpack_id?}
    U->>API: POST /servers/{id}/deploy
    API->>VPS: SSH: upload qx-agent binary
    API->>VPS: SSH: write systemd unit + env
    API->>VPS: SSH: systemctl enable --now qx-agent
    A->>API: WSS connect + auth
    API-->>U: status: online
```

### 2.2 Файлы на VPS

| Path | Назначение |
| ------ | ------------ |
| `/opt/qx/agent/qx-agent` | Binary |
| `/opt/qx/server/` | Server root (jar, mods, configs) |
| `/etc/qx/agent.env` | `QX_AGENT_TOKEN`, `QX_API_URL`, `QX_SERVER_ID` |
| `/etc/systemd/system/qx-agent.service` | systemd unit |

### 2.3 systemd unit (шаблон)

```ini
[Unit]
Description=QX Minecraft Agent
After=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/qx/agent.env
ExecStart=/opt/qx/agent/qx-agent
Restart=always
RestartSec=5
WorkingDirectory=/opt/qx/server

[Install]
WantedBy=multi-user.target
```

### 2.4 SSH требования

- Backend хранит `private_key_enc` (AES-GCM, master key из env).
- Supported: Ed25519, RSA keys.
- Minimum: user with sudo for `/opt/qx`, systemd.
- Firewall: исходящий HTTPS/WSS к QXApi (443).

---

## 3. WebSocket Connection

### 3.1 Handshake

```text
GET /agent/v1/connect HTTP/1.1
Upgrade: websocket
Authorization: Bearer <agent_jwt>
X-Agent-Version: 1.0.0
X-Server-Id: <uuid>
```

**Agent JWT claims:**

```json
{
  "sub": "agent:<server_uuid>",
  "server_id": "<uuid>",
  "exp": 1735689600
}
```

### 3.2 Reconnect

| Параметр | Значение |
| ---------- | ---------- |
| Initial backoff | 1s |
| Max backoff | 60s |
| Jitter | ±20% |
| Heartbeat interval | 30s |
| Heartbeat timeout | 90s (server closes) |

При reconnect agent **не** перезапускает MC process. Backend re-associates WSS session with `server_id`.

### 3.3 Idempotency

Каждая **command** содержит `request_id` (UUID v4).

- Agent хранит cache последних **1000** `request_id` (TTL 24h).
- Повтор command с тем же `request_id` → agent отвечает cached `*.result` без re-execution.
- Backend может safely retry on timeout (30s default).

---

## 4. Message Envelope

```typescript
interface Envelope {
  v: 1;
  type: string;
  request_id?: string;   // required for commands & results
  ts: string;            // ISO8601
  payload: unknown;
}
```

**Direction prefix:**

| Prefix | Direction |
| -------- | ----------- |
| `cmd.*` | Backend → Agent |
| `evt.*` | Agent → Backend |
| `res.*` | Agent → Backend (response to cmd) |

---

## 5. Commands (Backend → Agent)

### 5.1 Lifecycle

```json
{
  "v": 1,
  "type": "cmd.server.start",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "ts": "2026-06-09T12:00:00Z",
  "payload": {
    "server_type": "paper",
    "jar_path": "/opt/qx/server/server.jar",
    "jvm_args": ["-Xms2G", "-Xmx4G"],
    "extra_args": ["--nogui"]
  }
}
```

| type | payload |
| ------ | --------- |
| `cmd.server.start` | `server_type`, `jar_path`, `jvm_args[]`, `extra_args[]` |
| `cmd.server.stop` | `{ "graceful": true, "timeout_sec": 30 }` |
| `cmd.server.restart` | same as start + stop |
| `cmd.server.kill` | `{}` |

**server_type:** `vanilla` \| `paper` \| `spigot` \| `purpur` \| `forge` \| `neoforge` \| `fabric` \| `quilt` \|
`hybrid` + `hybrid_platform` optional.

### 5.2 Console & RCON

| type | payload |
| ------ | --------- |
| `cmd.console.input` | `{ "line": "say hello" }` |
| `cmd.rcon.command` | `{ "command": "list", "password_from_config": true }` |

### 5.3 Files

| type | payload |
| ------ | --------- |
| `cmd.files.list` | `{ "path": "plugins" }` |
| `cmd.files.read` | `{ "path": "server.properties", "max_bytes": 1048576 }` |
| `cmd.files.write` | `{ "path": "...", "content_base64": "..." }` |
| `cmd.files.upload` | `{ "path": "...", "url": "https://presigned..." }` |
| `cmd.files.delete` | `{ "path": "..." }` |

Paths relative to server root. Traversal blocked (`..`, absolute paths).

### 5.4 Modpack

```json
{
  "v": 1,
  "type": "cmd.modpack.install",
  "request_id": "...",
  "payload": {
    "modpack_id": "uuid",
    "manifest_url": "https://api.../modpacks/{id}/manifest",
    "manifest_sha256": "abc...",
    "wipe_existing": false
  }
}
```

Agent downloads manifest, verifies hash matches server record, installs to server root (mods/, config/, jar if
server-side modpack).

### 5.5 Agent maintenance

| type | payload |
| ------ | --------- |
| `cmd.agent.update` | `{ "binary_url": "...", "sha256": "..." }` |
| `cmd.agent.ping` | `{}` |

---

## 6. Events & Responses (Agent → Backend)

### 6.1 Heartbeat

```json
{
  "v": 1,
  "type": "evt.agent.heartbeat",
  "ts": "2026-06-09T12:00:30Z",
  "payload": {
    "cpu_percent": 12.5,
    "ram_used_mb": 4096,
    "ram_total_mb": 8192,
    "disk_free_mb": 50000,
    "uptime_sec": 86400,
    "agent_version": "1.0.0"
  }
}
```

### 6.2 Server status

```json
{
  "v": 1,
  "type": "evt.server.status",
  "payload": {
    "status": "running",
    "pid": 12345,
    "exit_code": null
  }
}
```

`status`: `stopped` \| `starting` \| `running` \| `stopping` \| `crashed`

### 6.3 Console stream

```json
{
  "v": 1,
  "type": "evt.console.output",
  "payload": {
    "stream": "stdout",
    "line": "[12:00:01 INFO]: Done!"
  }
}
```

`stream`: `stdout` \| `stderr` \| `rcon`

### 6.4 Metrics

```json
{
  "v": 1,
  "type": "evt.metrics",
  "payload": {
    "tps": 20.0,
    "players_online": 3,
    "players_max": 20,
    "player_list": ["Steve", "Alex"]
  }
}
```

### 6.5 Command result

```json
{
  "v": 1,
  "type": "res.files.list",
  "request_id": "550e8400-...",
  "payload": {
    "ok": true,
    "entries": [{ "name": "server.jar", "size": 123, "is_dir": false }]
  }
}
```

Error:

```json
{
  "payload": {
    "ok": false,
    "error": { "code": "PATH_DENIED", "message": "..." }
  }
}
```

| Error code | Meaning |
| ------------ | --------- |
| `PATH_DENIED` | Outside server root |
| `FILE_TOO_LARGE` | Exceeds max_bytes |
| `PROCESS_BUSY` | Start while running |
| `PROCESS_NOT_RUNNING` | Stop while stopped |
| `MODPACK_HASH_MISMATCH` | Manifest sha256 mismatch |
| `DOWNLOAD_FAILED` | Network/hash error |

---

## 7. Backend Routing (Agent Hub)

```mermaid
flowchart TB
    Panel[Panel WS console] --> API[API Agent Hub]
    API --> Redis[(Redis pub/sub)]
    Redis --> API2[API instance 2]
    API --> Agent[Agent WSS]
```

- Map: `server_id → websocket.Conn`
- Panel subscribes: `WS /servers/{id}/console`
- Hub forwards `evt.console.output` → panel; `cmd.console.input` → agent
- Multi-admin: RBAC check `server_members` before forward

---

## 8. Security

| Rule | Implementation |
| ------ | ---------------- |
| Agent token rotation | On redeploy; old token revoked |
| SSH keys encrypted at rest | AES-GCM in MySQL |
| File sandbox | chroot-like prefix `/opt/qx/server` |
| Command RBAC | owner/admin: all; viewer: console read-only |
| TLS | WSS only in production |

---

## 9. Versioning

| Field | Policy |
| ------- | -------- |
| `v` in envelope | Breaking change → increment |
| `X-Agent-Version` | semver; backend may reject outdated |

---

*См. также: [api.md](./api.md), [schema.sql](./schema.sql)*
