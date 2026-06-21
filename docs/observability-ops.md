# Observability & Operations

> **I9:** Pure **self-hosted**, no Cloudflare. **I8:** VPS TBD.  
> **Dev:** `infra/docker/docker-compose.yml` — MySQL, Redis, MinIO; API/web локально. **Flow C:** `make dev-vps-up`.  
> **Prod:** 🔲 не готов — [mvp §7.1](./mvp.md).

---

## 1. Stack (Self-Hosted Tier 0)

| Tool | Role |
| ------ | ------ |
| **Nginx** | TLS termination, static, rate limit |
| **Uptime Kuma** | HTTP/TCP checks, status page |
| **Netdata** (optional) | VPS metrics |
| **MySQL** | App + audit logs |
| **MinIO** | Launcher releases, server backups, skins |

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
| `https://api.qx.example.com/api/v1/health` | 60s (liveness) |
| `https://api.qx.example.com/api/v1/health/ready` | 60s (readiness) |
| `https://qx.example.com` | 60s |
| MySQL TCP | 5m |
| Disk > 85% | daily script |

Notify: Telegram bot / email (SMTP post-MVP).

---

## 4. Runbooks

### 4.1 API down

1. `docker compose ps`
2. `docker compose logs api --tail 100`
3. Restart: `docker compose restart api`
4. If MySQL: check connections

### 4.2 Restore MySQL

```bash
mysql -u qx -p qx < backup.sql
```

Weekly `mysqldump` via cron → Restic offsite.

### 4.3 Agent mass disconnect

1. Check Redis / API logs
2. Verify TLS cert expiry
3. Notify users status page

### 4.4 Master key rotation

See [security-legal.md §3.2](./security-legal.md)

---

## 5. Health endpoints

```text
GET /api/v1/health        → 200 ok (liveness)
GET /api/v1/health/ready  → DB ping (Phase 0); + Redis + MinIO — prod
```

---

## 6. Backups

| What | Schedule | Where |
| ------ | ---------- | ------- |
| MySQL | Daily 03:00 | Restic → second VPS / NAS |
| MinIO | Daily incremental | same |
| `.env`, nginx | on change | private git |

---

## 7. Pure self-hosted networking

- Single or dual VPS (I8 TBD provider/region)
- DNS A record → VPS IP (registrar DNS, no CF proxy)
- DDoS mitigation: Nginx limits + fail2ban only

---

*ADR: [0009](./adr/0009-pure-self-hosted.md)* · Prod 🔲 [mvp §7.1](./mvp.md)

Последнее обновление: 2026-06-10
