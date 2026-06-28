# SSH Deploy — Agent Provisioning

> **F7:** Backend SSH → Linux VPS → systemd agent.
> Security: [security-legal.md §3](./security-legal.md)
> **Конфиг (dev):** [configuration.md](./configuration.md) — `public_api_url`, `agent_binary_path` в `qxapi.toml`
> **Статус:** ✅ Phase 2. REST: base `/api/v1` (пути ниже относительные).

---

## 1. Overview

```mermaid
sequenceDiagram
    participant U as Admin (Panel)
    participant API as Backend
    participant D as SSH Deployer
    participant VPS as Linux VPS
    participant A as QXAgent

    U->>API: POST /api/v1/servers + ssh_credentials
    U->>API: POST /api/v1/servers/{id}/deploy
    API->>D: Run deploy (sync in request)
    D->>VPS: SSH connect (key from DB decrypted)
    D->>VPS: Upload agent binary
    D->>VPS: Write agent.toml + systemd unit
    D->>VPS: systemctl enable + restart qx-agent
    A->>API: WSS connect
    D->>API: deploy success
    API-->>U: agent_online true (MC still offline)
```

После deploy сервер **не** считается «Minecraft online» — только `agent_online`. Статус MC (`minecraft_running`, `status`) меняется после `POST …/start` или heartbeat с `mc_pid`.

---

## 2. Prerequisites (user VPS)

| Requirement | Detail |
| ------------- | -------- |
| OS | Linux x86_64 (Ubuntu 22.04+, Debian 12+) |
| User | sudo without password **or** root (discouraged) |
| SSH | Port 22 or custom, key-based auth |
| Firewall | Outbound **443** (or dev HTTP) to QXApi; inbound MC port user-defined |
| Allowlist | Optional: QX platform egress IP in `ufw` |

Panel shows **pre-flight checklist** before deploy.

**Dev VPS:** `make dev-vps-up` — контейнер `qx-vps-dev`, SSH `:2222`. В `qxapi.toml`: `public_api_url = "http://host.docker.internal:3000"`.

**Prod:** `QX_PUBLIC_API_URL=https://mc.qx-dev.ru` в `.env.prod` — [production-deploy.md](./production-deploy.md).

---

## 3. Deploy worker

Go SSH deployer (`internal/deploy/ssh_deployer.go`):

| Config | Value |
| -------- | ------- |
| SSH timeout | 30s connect, 10 min total |
| Retry | handled at API layer |

### 3.1 Whitelisted remote commands

No user-supplied shell. Deployer executes fixed script template:

1. `mkdir -p /opt/qx/agent /opt/qx/server` (`plugins/`, `mods/` — по [server-content-install.md](./server-content-install.md))
2. `install -m 755` agent binary to `/opt/qx/agent/qx-agent`
3. Write `/etc/qx-agent/agent.toml` (0600 root)
4. Write `/etc/systemd/system/qx-agent.service`
5. `systemctl daemon-reload && systemctl enable qx-agent && systemctl restart qx-agent`

**Re-deploy:** `restart` (не только `enable --now`) — новый `agent_token` подхватывается без ручного restart на VPS.

---

## 4. Credentials

Stored in `ssh_credentials` — see [schema.sql](./schema.sql).

User provides in panel:

```json
{
  "host": "203.0.113.10",
  "port": 22,
  "username": "ubuntu",
  "private_key_pem": "-----BEGIN OPENSSH PRIVATE KEY-----..."
}
```

API validates key format, **never** returns private key in GET.

---

## 5. Agent token issuance

On successful deploy:

1. Generate `agent_jwt` scoped to `server_id`
2. Write to `/etc/qx-agent/agent.toml`:

   ```toml
   api_base_url = "https://mc.qx-dev.ru/api/v1"
   server_id = "uuid"
   agent_token = "eyJ..."
   server_root = "/opt/qx/server"
   ```

3. Store `agent_token_hash` in `servers` table

**Re-deploy:** revokes old token, issues new, **restarts** systemd unit.

**Dev localhost SSH:** `agent_api_url` в deploy → `http://host.docker.internal:3000/api/v1` (см. `internal/deploy/agent_api_url.go`).

---

## 6. Failure handling

| Error | User message | Audit |
| ------- | -------------- | ------- |
| SSH auth failed | Check key and username | `server.deploy.failed` |
| Timeout | Firewall blocking SSH | same |
| systemctl failed | View deploy log in panel | same |
| Agent no WSS 5min | Check outbound HTTPS to API | `server.deploy.partial` |

Deploy log stored in deploy response / server detail (last error).

---

## 7. Modpack on deploy (optional)

If `server.modpack_id` set:

1. Agent connects
2. API sends `cmd.modpack.install`
3. Server waits before first `server.start`

---

## 8. Multi-admin

Only `owner` and `admin` may trigger deploy. `viewer` — read-only.

Audit includes `actor_id` for every deploy.

---

*См. [agent-protocol.md](./agent-protocol.md), [configuration.md](./configuration.md)*

Последнее обновление: 2026-06-21 (Phase 2 ✅, TOML config)
