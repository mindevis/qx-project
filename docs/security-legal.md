# Security & Legal

> Rate limiting, audit, encryption, compliance.
> Self-hosted **без Cloudflare** — [ADR-0009](./adr/0009-pure-self-hosted.md)
> **Phase 0–3:** JWT + bcrypt ✅; Redis rate limit и audit log — post-MVP / prod.  
> Пути rate limit ниже — относительно REST base `/api/v1`.

---

## 1. Rate limiting

Implementation: **Redis sliding window** + Gin middleware.

| Endpoint / action | Limit | Window | Key |
| ------------------- | ------- | -------- | ----- |
| `POST /auth/login` | 10 failures | 15 min | IP + email |
| `POST /auth/register` | 5 | 1 hour | IP |
| `POST /auth/guest` | 20 | 1 hour | IP |
| `POST /launcher/devices/register` | 10 | 1 hour | IP |
| `POST /launcher/devices/link` | 20 | 1 hour | IP |
| `POST /launcher/launch-requests` | 30 | 1 hour | user/device |
| `POST /servers/{id}/deploy` | 3 | 1 hour | user_id |
| Global API | 300 req | 1 min | IP |
| Authenticated API | 600 req | 1 min | user_id |

**Response:** `429 Too Many Requests` + `Retry-After` header.

```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Too many login attempts",
    "retry_after_sec": 900
  }
}
```

**Brute-force lockout:** after 10 failed logins → account cooldown 30 min (optional email notify post-MVP).

---

## 2. Audit log

Append-only table `audit_logs`. **Never delete** (retention 2 years, then archive to MinIO cold storage).

### 2.1 Events

| `action` | Trigger | Fields |
| ---------- | --------- | -------- |
| `auth.login` | Successful login | user_id, ip, user_agent |
| `auth.login_failed` | Failed login | email hash, ip |
| `auth.register` | Registration | user_id, ip |
| `device.register` | Tray first connect | device_id, ip |
| `device.link` | Site confirms link (HWID URL) | device_id, user/guest_id |
| `device.unlink` | Unlink | device_id, actor_id |
| `launch.request` | Play clicked | instance_id, device_id |
| `launch.start` | JVM spawned | instance_id, pid |
| `server.create` | New server | server_id |
| `server.deploy` | SSH deploy started | server_id, ssh_host |
| `server.deploy.success` | Agent online | server_id |
| `server.deploy.failed` | SSH error | server_id, error_code |
| `server.start` | Start command | server_id, actor_id |
| `server.stop` | Stop command | server_id, actor_id |
| `server.member.add` | Invite admin | server_id, target_user |
| `ssh.key.rotate` | User uploads new key | server_id |
| `skin.upload` | Skin changed | user_id |

### 2.2 Schema

```sql
CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    action      VARCHAR(64) NOT NULL,
    actor_id    UUID,
    actor_type  VARCHAR(16),  -- user, system, guest
    resource_type VARCHAR(32),
    resource_id UUID,
    metadata    JSONB NOT NULL DEFAULT '{}',
    ip          INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_actor ON audit_logs (actor_id, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_logs (resource_type, resource_id);
```

### 2.3 Access

- **Owner/admin:** audit for their servers only.
- **Platform admin:** full read (future role).

---

## 3. SSH key encryption & rotation

### 3.1 At rest

```text
ciphertext = AES-256-GCM(plaintext_key, DEK)
DEK = HKDF(master_key, server_id)
master_key = qxapi.toml ssh_master_key (dev) / `.env.prod` SSH_MASTER_KEY (prod, 32 bytes base64)
```

- Master key **only** in `qxapi.toml` (dev) / `infra/docker/.env.prod` (prod) — never in DB or git. См. [production-deploy.md](./production-deploy.md).
- Per-server DEK derivation prevents bulk decrypt if one row leaked.

### 3.2 Rotation procedure

| Step | Action |
| ------ | -------- |
| 1 | Generate new `ssh_master_key` (v2) in `qxapi.toml` (dev) or Docker secret (prod) — base64, 32 bytes |
| 2 | Run `qx-admin reencrypt-ssh-keys --from v1 --to v2` |
| 3 | Deploy API with v2, keep v1 read-only 24h |
| 4 | Revoke v1 |
| 5 | Audit: `ssh.master_key.rotated` |

**User key rotation:** user uploads new SSH key in panel → old ciphertext replaced → audit log.

### 3.3 SSH deploy security

- Backend connects from **fixed egress IP** (document in panel: «allow this IP in firewall»).
- Max session 10 min per deploy job.
- Commands whitelist: upload binary, write systemd, systemctl — no arbitrary shell from user input.
- See [ssh-deploy.md](./ssh-deploy.md).

---

## 4. Mojang EULA & offline/cracked

### 4.1 Position

QXSystem provides **tools** for Minecraft community. Users responsible for compliance with [Minecraft
EULA](https://www.minecraft.net/en-us/eula) and [Usage Guidelines](https://www.minecraft.net/en-us/usage-guidelines).

| Feature | Legal note |
| --------- | ------------ |
| **Microsoft OAuth** | Licensed play — compliant path |
| **Offline/Local profiles** | User choice; QX does not bypass Mojang auth for online servers with `online-mode=true` |
| **Cracked servers (`online-mode=false`)** | Server owner responsibility; QX provides panel toggle with **ToS acceptance** checkbox |
| **Premium (future)** | Must not sell Minecraft itself or capes/skins implying official Mojang goods |

### 4.2 User-facing requirements

**Registration:** checkbox «I agree to QX Terms and Minecraft third-party policies».

**Server create:** if `online_mode=false`:

> «You confirm you have rights to operate this server and comply with applicable laws and Mojang policies for
> third-party servers.»

### 4.3 Premium (future billing)

- Premium = **QX platform features** (more servers, backups, priority) — **not** Minecraft license.
- Do not bundle «free Minecraft» in marketing.
- Consult legal counsel before EU/RU launch with payments.

---

## 5. CurseForge ToS & direct client install

See also [modpacks-pipeline.md](./modpacks-pipeline.md), [ADR-0011](./adr/0011-client-local-content-install.md).

### 5.1 Allowed (typical CF API usage)

- Fetch metadata and **authorized download URLs** via API.
- **QXLauncher** downloads directly to user PC instance — **no re-hosting** mod/modpack binaries on MinIO.
- **QXAgent** downloads to BYOS server disk from same manifest URLs.

### 5.2 Restrictions

| Do | Don't |
| ---- | ------- |
| Use CF API download URLs per install | Mirror modpacks on public CDN / MinIO for redistribution |
| Attribute CurseForge as source | Strip mod author metadata |
| Respect API rate limits | Scrape without API key |
| Honor takedown requests | Keep serving removed projects |

### 5.3 Compliance checklist

- [ ] CurseForge API ToS accepted for production key
- [ ] `manifest.source` + `external_id` stored for audit
- [ ] TTL re-fetch metadata (24h) to detect removed projects
- [ ] DMCA/takedown contact in QX legal page

---

## 6. Two-factor authentication (2FA)

**Status:** post-MVP, optional.

| Phase | Scope |
| ------- | ------- |
| MVP | Email + password only |
| v2 | TOTP (Google Authenticator) optional per user |
| v2 | Require 2FA for server deploy / SSH key change (optional org setting) |

Storage: `users.totp_secret_enc` encrypted with master key.

---

## 7. Transport & headers (pure self-hosted)

No Cloudflare — all security on VPS:

| Control | Implementation |
| --------- | ---------------- |
| TLS 1.2+ | Let's Encrypt, Nginx |
| HSTS | `Strict-Transport-Security` max-age 31536000 |
| CSP | Strict policy on `/launcher` |
| CORS | `https://mc.qx-dev.ru` only (same origin для панели и API) |
| Cookies | `HttpOnly`, `Secure`, `SameSite=Lax` |
| JWT | Short access 15m, refresh 7d rotation |

**DDoS:** Nginx `limit_req`, fail2ban on SSH/API abuse. Accept higher risk vs Cloudflare — monitor bandwidth.

---

## 8. RBAC summary

| Resource | Guest (linked) | Registered | Premium (future) |
| ---------- | ---------------- | ------------ | ------------------ |
| Vanilla instance | ✓ | ✓ | ✓ |
| Mods/shaders/resource packs | ✗ | ✓ | ✓ |
| Modpacks | ✗ | ✓ (post-MVP) | ✓ |
| Skins upload | ✗ | ✓ (post-MVP) | ✓ |
| BYOS servers | ✗ | ✓ | ✓+limits |
| Server mods/plugins | ✗ | ✓ by `server_type` (post-MVP) | ✓ |
| Public server list | read | read | read |

---

## 9. Incident response (outline)

1. Detect (Uptime Kuma, logs, user report)
2. Contain (revoke tokens, block IP)
3. Audit log review
4. Notify affected users if credential leak
5. Post-mortem doc in `docs/incidents/`

---

*См. [ssh-deploy.md](./ssh-deploy.md), [launch-bridge.md](./launch-bridge.md), [server-content-install.md](./server-content-install.md), [configuration.md](./configuration.md)*

Последнее обновление: 2026-06-21 (HWID device link)
