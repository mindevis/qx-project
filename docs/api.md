# QXApi Specification

> **Версия:** **1.9** · Base URL: `https://mc.qx-dev.ru/api/v1` (dev: `http://localhost:3000/api/v1`) · **Prod:** ✅ live 2026-06-29
> Backend: **Go + Gin + GORM** · Код: `services/qxapi/`
> **Конфиг (dev):** [configuration.md](./configuration.md) · **Prod:** [production-deploy.md](./production-deploy.md)

Все REST-эндпоинты QXApi (включая health) живут под префиксом **`/api/v1`**.  
В таблицах ниже пути **относительные** к base URL (например `/auth/login` → `…/api/v1/auth/login`).  
**Исключение:** Agent Hub — `WS /agent/v1/connect` на корне хоста API, без `/api/v1`.

---

## 0. Implementation status

| Method | Path | Статус |
| -------- | ------ | -------- |
| POST | `/auth/register` | ✅ |
| POST | `/auth/login` | ✅ |
| POST | `/auth/refresh` | ✅ |
| POST | `/auth/guest` | 🔲 v2+ |
| POST | `/auth/logout` | ✅ |
| GET | `/users/me` | ✅ |
| GET | `/users/me/launcher-device` | ✅ |
| PATCH | `/users/me/password` | ✅ |
| PATCH | `/users/me/email` | ✅ |
| GET | `/users/me/mojang` | ✅ Microsoft/Minecraft link status |
| POST | `/users/me/mojang/oauth/start` | ✅ Start OAuth (returns `authorization_url`) |
| GET | `/mojang/oauth/callback` | ✅ OAuth callback (redirect to web profile) |
| DELETE | `/users/me/mojang` | ✅ Unlink Microsoft account |
| GET | `/health` | ✅ |
| GET | `/health/ready` | ✅ |
| POST | `/launcher/devices/register` | ✅ |
| GET | `/launcher/devices/:id/status` | ✅ |
| POST | `/launcher/devices/link` | ✅ |
| POST | `/launcher/devices/unlink` | ✅ device JWT |
| GET | `/launcher/devices/me` | ✅ device JWT |
| GET | `/launcher/devices/me/instances` | ✅ device JWT |
| GET | `/instances` | ✅ Bearer |
| POST | `/instances` | ✅ |
| GET | `/instances/:id` | ✅ |
| GET | `/instances/:id/manifest` | ✅ |
| DELETE | `/instances/:id` | ✅ |
| GET | `/launcher/profiles` | ✅ |
| POST | `/launcher/profiles` | ✅ |
| DELETE | `/launcher/profiles/:id` | ✅ |
| POST | `/launcher/launch-requests` | ✅ |
| GET | `/launcher/launch-requests/:id` | ✅ |
| GET | `/launcher/launch-requests/pending` | ✅ device JWT |
| PATCH | `/launcher/launch-requests/:id` | ✅ device JWT |
| GET/POST/DELETE | `/servers` … | ✅ Phase 2 |
| GET/PATCH/… | `/servers/:id/game-servers/…` | ✅ |
| POST | `/servers/:id/deploy|start|stop|restart` | ✅ |
| GET | `/servers/:id/console` | ✅ WebSocket |
| WS | `/agent/v1/connect` | ✅ Agent Hub |
| Skins, billing, public servers | — | 🔲 post-MVP |

**Ответ токенов (register / login / refresh):**

```json
{
  "access_token": "…",
  "refresh_token": "…",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**Ответ guest** *(v2+, endpoint не в router):*

```json
{
  "guest_token": "…",
  "expires_in": 86400
}
```

**Профиль (`GET /users/me`):**

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "username": "optional",
  "tier": "free",
  "created_at": "2026-06-10T12:00:00Z"
}
```

CORS: `cors_origin` в `qxapi.toml` (default `http://localhost:5173`). Полный список ключей: [configuration.md](./configuration.md).

### Health (Phase 0 ✅)

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/health` | — | Liveness (`{"status":"ok"}`) |
| GET | `/health/ready` | — | Readiness (DB ping) |

Полные URL: `GET {base}/health`, `GET {base}/health/ready`.

---

## 1. Auth

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| POST | `/auth/register` | — | `{ email, password, username? }` |
| POST | `/auth/login` | — | `{ email, password }` → `{ access_token, refresh_token }` |
| POST | `/auth/refresh` | refresh cookie/body | New access token |
| POST | `/auth/guest` | — | 🔲 v2+ `{ device_id }` → `{ guest_token }` |
| POST | `/auth/logout` | Bearer | Revoke refresh |

**Headers:** `Authorization: Bearer <access_token>`

---

## 2. Users & Skins (registered only)

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/users/me` | Bearer | Profile |
| GET | `/users/me/launcher-device` | Bearer | Linked QXLauncher device (`{ linked: false }` or device info) |
| PATCH | `/users/me/password` | Bearer | Change password (Phase 0 ✅) |
| PATCH | `/users/me/email` | Bearer | Change email (Phase 0 ✅) |
| PATCH | `/users/me` | Bearer | Update profile (name, etc.) — 🔲 post-MVP |
| POST | `/users/me/skin` | Bearer | Upload skin PNG (max 64KB) |
| DELETE | `/users/me/skin` | Bearer | Reset skin |
| GET | `/skins/{uuid}.png` | — | Public skin texture |
| GET | `/capes/{uuid}.png` | — | Public cape texture |

> Offline Local profiles: skins **not** uploaded; use default Steve/Alex.

---

## 3. Instances (client)

Требуется **Bearer JWT** (зарегистрированный пользователь) + привязанный `device_token` для launch.  
Modpacks и загрузка mods/shaders/RP из панели — v2+. См. [security-legal.md §8](./security-legal.md).

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/instances` | Bearer | List instances |
| POST | `/instances` | Bearer | Create `{ name, mc_version, loader, loader_version?, modpack_id? }` — `loader`: `vanilla` \| `forge` \| `neoforge` \| `fabric` \| `quilt` |
| GET | `/instances/{id}` | Bearer | Detail |
| PATCH | `/instances/{id}` | Bearer | Update — **post-MVP** (not implemented) |
| DELETE | `/instances/{id}` | Bearer | Delete |
| GET | `/instances/{id}/manifest` | Bearer | Launch manifest (QXLauncher) |

---

## 4. Servers (Dedicated)

### 4.1 Lifecycle

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/servers` | Bearer | List (owned + member) |
| POST | `/servers` | Bearer | Create `{ name, server_type, hybrid_platform?, ssh, modpack_id? }` |
| GET | `/servers/{id}` | Bearer | Detail + status |
| PATCH | `/servers/{id}` | Bearer | Update config |
| DELETE | `/servers/{id}` | Bearer | owner only |
| POST | `/servers/{id}/deploy` | Bearer | SSH deploy agent (admin+) |
| POST | `/servers/{id}/start` | Bearer | admin+ |
| POST | `/servers/{id}/stop` | Bearer | admin+ |
| POST | `/servers/{id}/restart` | Bearer | admin+ |
| GET | `/servers/{id}/console` | Bearer (WS) | Live console; `?game_server_id=` для фильтра |

**GET `/servers/{id}` response (status fields):**

| Field | Type | Description |
| ------- | ------ | ------------- |
| `agent_online` | bool | QXAgent WSS connected |
| `minecraft_running` | bool | JAR process running (`mc_pid` set) |
| `status` | string | MC lifecycle: `offline`, `starting`, `online`, `error` |
| `config.mc_pid` | int? | PID on dedicated server (when running) |

### 4.2 Game servers (per dedicated server)

Игровые инстансы Minecraft на выделенном сервере с QXAgent. UI: таблица на странице выделенного сервера → `/servers/:serverId/game-servers/:id`.

| Method | Path | Description |
| -------- | ------ | ------------- |
| GET | `/servers/{id}/game-servers` | List |
| POST | `/servers/{id}/game-servers` | Create `{ name, server_type, mc_version, loader_version?, address?, port? }` |
| GET | `/servers/{id}/game-servers/{gameServerId}` | Detail |
| PATCH | `/servers/{id}/game-servers/{gameServerId}` | Update name/address/port |
| DELETE | `/servers/{id}/game-servers/{gameServerId}` | Delete |
| POST | `…/reinstall` | Reinstall core |
| POST | `…/start` \| `…/stop` \| `…/restart` | Power |
| GET | `…/properties` | Read `server.properties` (agent RPC) |
| PATCH | `…/properties` | `{ updates: { key: value } }` |
| GET | `…/mods` | List mods/plugins folder |
| GET | `…/files?path=` | List directory |
| GET | `…/files/content?path=` | Read file |
| PUT | `…/files/content?path=` | `{ content }` — write file |

**Game server status:** `installing` | `starting` | `running` | `stopped` | `error`

Deploy/onboarding: [production-deploy.md §7](./production-deploy.md) · [ssh-deploy.md](./ssh-deploy.md).

### 4.3 Server content (mods / plugins) — post-MVP API

По `server_type` — см. [post-mvp.md#server-content](./post-mvp.md#server-content).  
Ошибка `403 CONTENT_NOT_ALLOWED` если тип не поддерживает контент.

| Method | Path | Auth | Allowed `server_type` |
| -------- | ------ | ------ | ----------------------- |
| POST | `/servers/{id}/mods` | Bearer admin+ | forge, neoforge, fabric, quilt, hybrid |
| POST | `/servers/{id}/plugins` | Bearer admin+ | paper, spigot, purpur, hybrid |
| POST | `/servers/{id}/modpack` | Bearer admin+ | compatible loader |
| GET | `/servers/{id}/content` | Bearer admin+ | List installed mods/plugins metadata |

### 4.4 Server members (multi-admin) — post-MVP

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/servers/{id}/members` | Bearer | admin+ |
| POST | `/servers/{id}/members` | Bearer | `{ user_id or email, role }` owner only |
| PATCH | `/servers/{id}/members/{uid}` | Bearer | Change role (owner) |
| DELETE | `/servers/{id}/members/{uid}` | Bearer | Remove member |

**Roles:** `owner` | `admin` | `viewer`

| Role | start/stop | files | console write | deploy | members |
| ------ | ------------ | ------- | --------------- | -------- | --------- |
| owner | ✓ | ✓ | ✓ | ✓ | ✓ |
| admin | ✓ | ✓ | ✓ | ✓ | read |
| viewer | — | read | read | — | — |

---

## 5. Modpacks

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/modpacks/search` | — / Bearer | `?q=&source=curseforge` (default CF, `modrinth` fallback) |
| GET | `/modpacks/{id}` | — | Metadata |
| GET | `/modpacks/{id}/manifest` | Bearer | QxModpackManifest JSON — 🔲 post-MVP |
| POST | `/modpacks/import` | Bearer | Import from CF/MR external id |

**Modpack sync:** same `modpack_id` on `instances` and `servers` → QXLauncher (ПК) + QXAgent (сервер).
На сервере состав путей по `server_type`: [post-mvp.md#server-content](./post-mvp.md#server-content).

---

## 6. Public servers (launcher UI)

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
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
| -------- | ------ | ------------- |
| POST | `/launcher/devices/register` | QXLauncher first launch — `{ device_id, os, hostname, launcher_version }`; `device_id` = HWID ПК |
| GET | `/launcher/devices/{id}/status` | Poll link status |
| POST | `/launcher/devices/link` | Web confirms link — `{ device_id }` + **Bearer JWT** |
| POST | `/launcher/devices/unlink` | Unlink device |
| GET | `/launcher/devices/me/instances` | `Bearer <device_token>` | Tray sync — instances for linked owner |
| GET | `/launcher/devices/me` | Current device (device_token) |

Full spec: [device-linking.md](./device-linking.md)

**Register response (201):**

```json
{
  "device_id": "uuid-from-hwid",
  "status": "pending_link",
  "link_url": "http://localhost:5173/launcher/link?device=uuid-from-hwid",
  "poll_interval_sec": 3,
  "expires_at": "2026-06-21T12:00:00Z"
}
```

QXLauncher открывает `link_url` в браузере автоматически. Поле `user_code` **не используется** (legacy в БД).

---

## 8. Launch requests (hybrid B1)

Site creates request → QXLauncher polls → spawns JVM. Full spec: [launch-bridge.md](./launch-bridge.md)

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| POST | `/launcher/launch-requests` | Bearer + `X-Device-Token` | Create `{ instance_id, offline_profile_id?, jvm_args_override? }` |
| GET | `/launcher/launch-requests/pending` | `Bearer <device_token>` | Tray poll — returns oldest queued, marks `dispatched` |
| PATCH | `/launcher/launch-requests/{id}` | `Bearer <device_token>` | Tray update `{ status, pid?, exit_code?, error? }` |
| GET | `/launcher/launch-requests/{id}` | Bearer | UI status poll |

**Auth:** Bearer JWT пользователя + заголовок `X-Device-Token` при создании запроса. Guest — v2+.

**Statuses:** `queued` → `dispatched` → `running` → `completed` | `failed` | `expired` (TTL 5 min)

---

## 9. Launcher updates (tray)

| Method | Path | Auth | Description |
| -------- | ------ | ------ | ------------- |
| GET | `/launcher/updates/latest` | — | `{ version, url, sha256, mandatory }` |
| GET | `/launcher/updates/latest.yml` | — | Electron-style (optional) |

**QXLauncher** polls update endpoint on startup. **UI** — QXWeb static на `https://mc.qx-dev.ru/launcher` (не WebView,
не через API).

---

## 10. WebSocket

| Path | Auth | Description |
| ------ | ------ | ------------- |
| `WS /api/v1/servers/{id}/console?access_token=…` | User JWT (query **or** `Authorization: Bearer`) | Live console (admin+ read/write by role) |
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
  title: QXSystem API
  version: 1.0.0
servers:
  - url: https://mc.qx-dev.ru/api/v1
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

> Full OpenAPI: **Swagger UI** at `http://localhost:3000/swagger/index.html` when QXApi is running.  
> Regenerate from annotations: `make swagger` → `services/qxapi/docs/swagger.{json,yaml}`.

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
| ------ | ------ |
| 400 | VALIDATION_ERROR |
| 401 | UNAUTHORIZED |
| 403 | FORBIDDEN, CONTENT_NOT_ALLOWED |
| 404 | NOT_FOUND |
| 409 | CONFLICT |
| 422 | HOST_NOT_LINUX |
| 429 | RATE_LIMITED |
| 500 | INTERNAL |

---

*См. [schema.sql](./schema.sql), [agent-protocol.md](./agent-protocol.md), [security-legal.md](./security-legal.md), [configuration.md](./configuration.md)*

---

Последнее обновление: 2026-06-21 (v1.9 — HWID device link, без user_code)
