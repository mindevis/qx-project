# Observability & Operations

> **I9:** Pure **self-hosted**, no Cloudflare. **I8:** VPS TBD.

---

## 1. Stack (Self-Hosted Tier 0)

| Tool | Role |
| ------ | ------ |
| **Nginx** | TLS termination, static, rate limit |
| **Uptime Kuma** | HTTP/TCP checks, status page |
| **Netdata** (optional) | VPS metrics |
| **PostgreSQL** | App + audit logs |
| **MinIO** | Files, releases, modpack cache |

No Prometheus in MVP — add at Tier 1.

---

## 2. Logging

Structured JSON logs from Go (`slog`):

```json
{
  "level": "info",
  "msg": "deploy completed",
  "server_id": "uuid",
  "duration_ms": 12400,
  "request_id": "..."
}
```

| Destination | Retention |
| ------------- | ----------- |
| stdout → Docker | 7 days rotate |
| `audit_logs` table | 2 years |

---

## 3. Alerts (Uptime Kuma)

| Check | Interval |
| ------- | ---------- |
| `https://api.qx.example.com/health` | 60s |
| `https://qx.example.com` | 60s |
| PostgreSQL TCP | 5m |
| Disk > 85% | daily script |

Notify: Telegram bot / email (SMTP post-MVP).

---

## 4. Runbooks

### 4.1 API down

1. `docker compose ps`
2. `docker compose logs api --tail 100`
3. Restart: `docker compose restart api`
4. If PG: check connections

### 4.2 Restore PostgreSQL

```bash
pg_restore -d qx < backup.dump
```

Weekly `pg_dump` via cron → Restic offsite.

### 4.3 Agent mass disconnect

1. Check Redis / API logs
2. Verify TLS cert expiry
3. Notify users status page

### 4.4 Master key rotation

See [security-legal.md §3.2](./security-legal.md)

---

## 5. Health endpoints

```text
GET /health        → 200 ok
GET /health/ready  → DB + Redis + MinIO ping
```

---

## 6. Backups

| What | Schedule | Where |
| ------ | ---------- | ------- |
| PostgreSQL | Daily 03:00 | Restic → second VPS / NAS |
| MinIO | Daily incremental | same |
| `.env`, nginx | on change | private git |

---

## 7. Pure self-hosted networking

- Single or dual VPS (I8 TBD provider/region)
- DNS A record → VPS IP (registrar DNS, no CF proxy)
- DDoS mitigation: Nginx limits + fail2ban only

---

*ADR: [0009](./adr/0009-pure-self-hosted.md)*
