# SSH Deploy — Agent Provisioning

> **F7:** Backend SSH → Linux VPS → systemd agent.
> Security: [security-legal.md §3](./security-legal.md)

---

## 1. Overview

```mermaid
sequenceDiagram
    participant U as Admin (Panel)
    participant API as Backend
    participant Q as Deploy Worker
    participant VPS as Linux VPS
    participant A as QX Agent

    U->>API: POST /servers + ssh_credentials
    U->>API: POST /servers/{id}/deploy
    API->>Q: Enqueue deploy_job
    Q->>VPS: SSH connect (key from DB decrypted)
    Q->>VPS: Upload agent binary
    Q->>VPS: Write env + systemd unit
    Q->>VPS: systemctl enable --now
    A->>API: WSS connect
    Q->>API: job success + audit
    API-->>U: server status online
```

---

## 2. Prerequisites (user VPS)

| Requirement | Detail |
| ------------- | -------- |
| OS | Linux x86_64 (Ubuntu 22.04+, Debian 12+) |
| User | sudo without password **or** root (discouraged) |
| SSH | Port 22 or custom, key-based auth |
| Firewall | Outbound **443** to QX API; inbound MC port user-defined |
| Allowlist | Optional: QX platform egress IP in `ufw` |

Panel shows **pre-flight checklist** before deploy.

---

## 3. Deploy worker

Go goroutine pool (`internal/deploy/worker.go`):

| Config | Value |
| -------- | ------- |
| Max concurrent deploys | 5 global, 1 per server |
| SSH timeout | 30s connect, 10 min total |
| Retry | 2 retries exponential backoff |

### 3.1 Whitelisted remote commands

No user-supplied shell. Worker executes fixed script template:

1. `mkdir -p /opt/qx/agent /opt/qx/server`
2. `install -m 755` agent binary to `/opt/qx/agent/qx-agent`
3. Write `/etc/qx/agent.env` (0600 root)
4. Write `/etc/systemd/system/qx-agent.service`
5. `systemctl daemon-reload && systemctl enable --now qx-agent`

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
2. Write to `/etc/qx/agent.env`:

   ```env
   QX_API_URL=https://api.qx.example.com
   QX_SERVER_ID=uuid
   QX_AGENT_TOKEN=eyJ...
   QX_SERVER_ROOT=/opt/qx/server
   ```

3. Store `agent_token_hash` in `servers` table

**Re-deploy:** revokes old token, issues new.

---

## 6. Failure handling

| Error | User message | Audit |
| ------- | -------------- | ------- |
| SSH auth failed | Check key and username | `server.deploy.failed` |
| Timeout | Firewall blocking SSH | same |
| systemctl failed | View deploy log in panel | same |
| Agent no WSS 5min | Check outbound HTTPS | `server.deploy.partial` |

Deploy log stored in `deploy_jobs.log` (last 64 KB).

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

*См. [agent-protocol.md](./agent-protocol.md)*
