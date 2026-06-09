# QX API Specification

> Версия: **1.0** · Base URL: `https://api.qx.example.com/v1`  
> Backend: **Go + Gin + GORM**

---

## 1. Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | — | `{ email, password, username? }` |
| POST | `/auth/login` | — | `{ email, password }` → `{ access_token, refresh_token }` |
| POST | `/auth/refresh` | refresh cookie/body | New access token |
| POST | `/auth/guest` | — | `{ device_id }` → `{ guest_token }` |
| POST | `/auth/logout` | Bearer | Revoke refresh |

**Headers:** `Authorization: Bearer <access_token>`

---

## 2. Users & Skins (registered only)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/users/me` | Bearer | Profile |
| PATCH | `/users/me` | Bearer | Update profile |
| POST | `/users/me/skin` | Bearer | Upload skin PNG (max 64KB) |
| DELETE | `/users/me/skin` | Bearer | Reset skin |
| GET | `/skins/{uuid}.png` | — | Public skin texture |
| GET | `/capes/{uuid}.png` | — | Public cape texture |

> Guest/offline Local accounts: skins **not** uploaded; use default.

---

## 3. Instances (client)

RBAC: **Guest** — Vanilla only, no mods/shaders/resource packs. **Registered** — full loaders + attachments. См. [security-legal.md §8](./security-legal.md).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/instances` | Bearer / Guest | List instances |
| POST | `/instances` | Bearer / Guest | Create `{ name, mc_version, loader, modpack_id? }` — Guest: `loader=vanilla` only |
| GET | `/instances/{id}` | Bearer / Guest | Detail |
| PATCH | `/instances/{id}` | Bearer / Guest | Update — Guest: no mod attachments |
| DELETE | `/instances/{id}` | Bearer / Guest | Delete |
| GET | `/instances/{id}/manifest` | Bearer / Guest | Launch manifest (Go tray) |

---

## 4. Servers (BYOS)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/servers` | Bearer | List (owned + member) |
| POST | `/servers` | Bearer | Create `{ name, server_type, ssh, modpack_id? }` |
| GET | `/servers/{id}` | Bearer | Detail + status |
| PATCH | `/servers/{id}` | Bearer | Update config |
| DELETE | `/servers/{id}` | Bearer | owner only |
| POST | `/servers/{id}/deploy` | Bearer | SSH deploy agent (admin+) |
| POST | `/servers/{id}/start` | Bearer | admin+ |
| POST | `/servers/{id}/stop` | Bearer | admin+ |
| POST | `/servers/{id}/restart` | Bearer | admin+ |

### 4.1 Server members (multi-admin)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/servers/{id}/members` | Bearer | admin+ |
| POST | `/servers/{id}/members` | Bearer | `{ user_id or email, role }` owner only |
| PATCH | `/servers/{id}/members/{uid}` | Bearer | Change role (owner) |
| DELETE | `/servers/{id}/members/{uid}` | Bearer | Remove member |

**Roles:** `owner` | `admin` | `viewer`

| Role | start/stop | files | console write | deploy | members |
|------|------------|-------|---------------|--------|---------|
| owner | ✓ | ✓ | ✓ | ✓ | ✓ |
| admin | ✓ | ✓ | ✓ | ✓ | read |
| viewer | — | read | read | — | — |

---

## 5. Modpacks

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/modpacks/search` | — / Bearer | `?q=&source=curseforge` (default CF, `modrinth` fallback) |
| GET | `/modpacks/{id}` | — | Metadata |
| GET | `/modpacks/{id}/manifest` | Bearer / Guest | QxModpackManifest JSON |
| POST | `/modpacks/import` | Bearer | Import from CF/MR external id |

**Modpack sync:** same `modpack_id` on `instances` and `servers` → client + agent install.

---

## 6. Public servers (launcher UI)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/public/servers` | — | Paginated public server list |
| GET | `/public/servers/{id}` | — | Detail for launcher card |

Query: `?page=&loader=&version=&q=`

Response item:

```json
{
  "id": "uuid",
  "name": "My Server",
  "description": "...",
  "mc_version": "1.20.4",
  "loader": "fabric",
  "online_players": 12,
  "max_players": 50,
  "is_online": true,
  "address": "play.example.com"
}
```

> **`/launcher` UI** on website fetches this; tray launches game via [launch-bridge](./launch-bridge.md).

---

## 7. Device Linking & Launcher Tray

| Method | Path | Description |
|--------|------|-------------|
| POST | `/launcher/devices/register` | Go tray first launch |
| GET | `/launcher/devices/{id}/status` | Poll link status |
| POST | `/launcher/devices/link` | Web confirms link (guest or user) |
| POST | `/launcher/devices/unlink` | Unlink device |
| GET | `/launcher/devices/me` | Current device (device_token) |

Full spec: [device-linking.md](./device-linking.md)

---

## 8. Launch requests (hybrid B1)

Site creates request → Go tray polls → spawns JVM. Full spec: [launch-bridge.md](./launch-bridge.md)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/launcher/launch-requests` | Bearer / Guest + `X-Device-Token` | Create `{ instance_id, offline_profile_id?, jvm_args_override? }` |
| GET | `/launcher/launch-requests/pending` | `Bearer <device_token>` | Tray poll — returns oldest queued, marks `dispatched` |
| PATCH | `/launcher/launch-requests/{id}` | `Bearer <device_token>` | Tray update `{ status, pid?, exit_code?, error? }` |
| GET | `/launcher/launch-requests/{id}` | Bearer / Guest | UI status poll |

**RBAC:** Guest — Vanilla instances only. Registered — all loaders + mods/shaders/resource packs.

**Statuses:** `queued` → `dispatched` → `running` → `completed` | `failed` | `expired` (TTL 5 min)

---

## 9. Launcher updates (tray)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/launcher/updates/latest` | — | `{ version, url, sha256, mandatory }` |
| GET | `/launcher/updates/latest.yml` | — | Electron-style (optional) |
| GET | `/launcher/ui/*` | — | Static SPA assets (or CDN) |

**Go tray** polls update endpoint on startup. **UI** at `https://qx.example.com/launcher` (React on site, no WebView).

---

## 10. WebSocket

| Path | Auth | Description |
|------|------|-------------|
| `WS /servers/{id}/console` | Bearer | Live console (admin+ read/write by role) |
| `WS /agent/v1/connect` | Agent JWT | Agent Hub ([agent-protocol.md](./agent-protocol.md)) |

### Console WS messages

```json
{ "type": "output", "stream": "stdout", "line": "..." }
{ "type": "input", "line": "say hello" }
{ "type": "status", "status": "running", "players_online": 3 }
```

---

## 11. OpenAPI stub

```yaml
openapi: 3.0.3
info:
  title: QXProject API
  version: 1.0.0
servers:
  - url: https://api.qx.example.com/v1
paths:
  /auth/login:
    post:
      summary: Login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email, password]
              properties:
                email: { type: string, format: email }
                password: { type: string, minLength: 8 }
      responses:
        '200':
          description: Tokens
          content:
            application/json:
              schema:
                type: object
                properties:
                  access_token: { type: string }
                  refresh_token: { type: string }
  /instances:
    get:
      security: [{ bearerAuth: [] }]
      summary: List instances
    post:
      security: [{ bearerAuth: [] }]
      summary: Create instance
  /servers/{id}/deploy:
    post:
      security: [{ bearerAuth: [] }]
      summary: SSH deploy agent
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

> Full OpenAPI export: `go run cmd/openapi/main.go` (TBD in repo).

---

## 12. Errors

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "email is required",
    "details": []
  }
}
```

| HTTP | code |
|------|------|
| 400 | VALIDATION_ERROR |
| 401 | UNAUTHORIZED |
| 403 | FORBIDDEN |
| 404 | NOT_FOUND |
| 409 | CONFLICT |
| 429 | RATE_LIMITED |
| 500 | INTERNAL |

---

*См. [schema.sql](./schema.sql), [agent-protocol.md](./agent-protocol.md), [security-legal.md](./security-legal.md)*
