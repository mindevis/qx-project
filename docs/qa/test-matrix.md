# QXSystem — QA Test Matrix (Alpha)

> REST base: `http://localhost:3000/api/v1` (prod: `https://mc.qx-dev.ru/api/v1`).  
> Пути в таблицах — относительные к base, если не указано иное.
>
> Для закрытой beta / MVP Definition of Done.
> Статус теста: ☐ не пройден · ☑ пройден · ⊘ N/A (post-MVP) · 🤖 автоматизирован (unit)
> Server content (mods/plugins): [post-mvp.md#server-content](../post-mvp.md#server-content)

**Flow A/B:** manual pass ☑ — A09, L03, I04, I05 (QXLauncher: auto browser link, tray fallback, full JVM, Mojang JRE с нуля).

**Flow C (dev dedicated server):** manual pass ☑ — S01, S02, S11. Prereq: `make dev-vps-up`, SSH `:2222`, `public_api_url` в `qxapi.toml` — см. [configuration.md](../configuration.md).

---

## 0. Automated unit tests (CI)

| Область | Покрытие | Команда | Статус |
| --------- | ---------- | --------- | -------- |
| `services/qxapi` | 100% Go statements | `cd services/qxapi && go test ./...` | 🤖 ☑ |
| `services/qxagent/cmd` | cmd + internal | `cd services/qxagent && go test ./...` | 🤖 ☑ |
| `services/qxlauncher/cmd` | cmd + internal + tray | `cd services/qxlauncher && go test ./...` | 🤖 ☑ |
| `web/qxweb` | 100% stmts/branches | `cd web/qxweb && npm run test:coverage` | 🤖 ☑ |
| E2E API smoke | Flow A/B/C router | `make e2e-api-smoke` | 🤖 ☑ |
| E2E launch-bridge dry-run | QXLauncher poll → dry JVM | `make e2e-dry-run` | 🤖 ☑ |
| E2E JVM smoke | Mojang manifest + `java -version` | `make e2e-jvm` | 🤖 ☑ · ☑ manual I05 |
| E2E (Playwright / manual QXLauncher+JVM) | A09, L03, I04 full MC client | `make e2e-alpha` + `make e2e-manual` | 🤖 ☑ · ☑ manual Flow A/B |

---

## 1. Auth & Users

| ID | Сценарий | Шаги | Ожидание | MVP | Статус |
| ---- | ---------- | ------ | ---------- | ----- | -------- |
| A01 | Register | email + password на panel | 201, login possible | ☑ | 🤖 ☑ |
| A02 | Login | valid credentials | JWT, redirect `/profile` | ☑ | 🤖 ☑ |
| A02b | Change password | PATCH `/users/me/password` | 204, login with new password | ☑ | 🤖 ☑ |
| A02c | Change email | PATCH `/users/me/email` | 200, updated profile | ☑ | 🤖 ☑ |
| A03 | Login fail | wrong password | 401 | ☑ | 🤖 ☑ |
| A04 | Guest token | POST /auth/guest | guest_token returned | ⊘ v2+ | ⊘ |
| A05 | Skin upload | auth user POST skin | visible GET /skins/{uuid} | ⊘ | ☐ |
| A07 | Device register | QXLauncher POST register (HWID `device_id`) | pending_link + link_url | ☑ | 🤖 ☑ `router_test` |
| A08 | Device link | web login + confirm + QXLauncher poll | linked + device_token | ☑ | 🤖 ☑ Flow A |
| A09 | Auto link page | Запуск QXLauncher | browser opens `/launcher/link?device=…` | ☑ | ☑ manual |

---

## 2. Instances (Client)

| ID | Сценарий | Шаги | Ожидание | MVP | Статус |
| ---- | ---------- | ------ | ---------- | ----- | -------- |
| I01 | Create instance auth | login + linked device + panel: loader + MC version | instance in list | ☑ | 🤖 ☑ Flow A |
| I02 | Create instance guest | guest session + web | instance linked to device | ⊘ v2+ | ⊘ |
| I03 | Launcher sync | open QXLauncher | instance appears | ☑ | 🤖 ☑ QXLauncher `syncInstances` |
| I04 | Launch Vanilla | Play button | MC client starts | ☑ | 🤖 ☑ · ☑ manual full MC |
| I05 | Mojang Java | fresh install (нет Java в PATH) | Mojang JRE downloaded | ☑ | 🤖 ☑ · ☑ manual |
| I06 | Modpack instance | select modpack | ⊘ post-MVP | ⊘ | ⊘ |
| I07 | Modded client launch | Forge/NeoForge/Fabric/Quilt | classpath + JVM start | ☑ | 🤖 ☑ QXLauncher unit/e2e |

---

## 3. Launcher UI (React on website)

| ID | Сценарий | MVP | Статус |
| ---- | ---------- | ----- | -------- |
| L01 | /launcher page loads | ☑ | ☑ instances + profiles UI |
| L02 | Device link flow | ☑ | 🤖 ☑ `LauncherLinkPage` tests |
| L03 | QXLauncher tray fallback | ПКМ «Связать QXLauncher» если браузер не открылся | ☑ | ☑ manual |
| L04 | Public servers tab | ⊘ | ⊘ |

---

## 4. Servers & Agent (Linux)

> **UI MVP:** после Deploy — тег **Agent** (`agent_online`); кнопки Stop/Restart и консоль — только при `minecraft_running`. Кнопка **Start** скрыта (post-MVP install pipeline). API `POST …/start|stop|restart` — 🤖 в router Flow C.

| ID | Сценарий | Шаги | Ожидание | MVP | Статус |
| ---- | ---------- | ------ | ---------- | ----- | -------- |
| S01 | Create server | panel + SSH creds | server pending | ☑ | 🤖 ☑ · ☑ manual dev |
| S02 | SSH deploy | Deploy agent | `agent_online`, systemd running | ☑ | 🤖 ☑ · ☑ manual dev dedicated server |
| S03 | Start server | API start или post-MVP UI | `minecraft_running`, status online | ☑ | 🤖 ☑ API · ⊘ UI Start |
| S04 | Stop server | Stop (при MC running) | process killed | ☑ | 🤖 ☑ API · ⊘ UI manual MC |
| S05 | Live console | WS console при MC running | stdout visible | ☑ | 🤖 ☑ · ⊘ UI manual MC |
| S06 | Console input | type command | executed in MC | ☑ | 🤖 ☑ · ⊘ UI manual MC |
| S07 | Multi-admin invite | add admin user | admin can start | ⊘ | ⊘ |
| S08 | Viewer read-only | viewer login | console read, no start | ⊘ | ⊘ |
| S09 | Modpack server sync | same modpack_id | agent installs by loader | ⊘ | ⊘ |
| S10 | Hybrid jar (Mohist) | server_type hybrid | mods/ + plugins/ | ⊘ | ⊘ |
| S10a | Plugins on Paper | POST /servers/{id}/plugins | plugins/ | ⊘ | ⊘ |
| S10b | Mods on NeoForge | POST /servers/{id}/mods | mods/ | ⊘ | ⊘ |
| S10c | Reject mods on Paper | POST /servers/{id}/mods | 403 CONTENT_NOT_ALLOWED | ⊘ | ⊘ |
| S10d | Reject plugins on NeoForge | POST /servers/{id}/plugins | 403 | ⊘ | ⊘ |
| S10e | Hybrid both content types | mods + plugins Mohist | both dirs | ⊘ | ⊘ |
| S11 | Agent reconnect | redeploy / restart API | agent reconnects ≤60s | ☑ | 🤖 ☑ · ☑ manual dev |
| S12 | Idempotent start | duplicate request_id | no double process | ☑ | 🤖 ☑ agent `requestCache` replay |
| S13 | Agent ≠ MC status | deploy only | `agent_online` true, `minecraft_running` false | ☑ | 🤖 ☑ · ☑ manual dev |

---

## 5. Modpack sync (post-MVP)

| ID | Сценарий | Ожидание | MVP |
| ---- | ---------- | ---------- | ----- |
| M01 | Same modpack_id client+server | hash match both sides | ⊘ |
| M02 | CurseForge import | manifest in DB | ⊘ |
| M03 | Modrinth mrpack | manifest in DB | ⊘ |

---

## 6. Infra & Non-functional

| ID | Сценарий | Ожидание | MVP | Статус |
| ---- | ---------- | ---------- | ----- | -------- |
| N01 | Docker Compose dev | MySQL, Redis, MinIO up | ☑ | ☑ `make dev-up` |
| N02 | TLS / prod reachability | HTTP 200, API health OK | ☑ | ☑ prod 2026-06-29 (`/api/v1/health`); HTTPS — после certbot |
| N03 | MySQL backup restore | data intact | ⊘ | ⊘ |
| N04 | Agent non-Linux | deploy to Windows SSH — fail gracefully | ☑ | 🤖 ☑ `ErrNonLinuxHost` |
| N05 | dev dedicated server Flow C | `make dev-vps-up`, deploy agent | agent WSS to API | ☑ | ☑ manual |

---

## 7. Regression checklist (pre-release)

**MVP alpha (dev)** — flows пройдены. **Prod platform** — ✅ smoke 2026-06-29.

- [x] Flow A/B — manual QXLauncher + full JVM (A09, L03, I04, I05)
- [x] Flow C deploy agent — manual dev dedicated server
- [x] **Prod readiness** — P.1–P.4, P.7 ([production-deploy.md](../production-deploy.md), [mvp §7.1](../mvp.md))
- [ ] P.5 бэкапы MySQL · P.2b HTTPS (certbot)
- [ ] All ☑ MVP rows passed (manual + E2E)
- [ ] All 🤖 unit tests green in CI
- [ ] No P0/P1 open bugs
- [ ] `docs/api.md` matches implemented routes
- [ ] `agent-protocol.md` message types match code (Phase 2+)
- [ ] Junior completed manual pass with notes

---

## 8. Prod readiness

Чеклист P.1–P.7: [mvp §7.1](../mvp.md#71-prod-readiness) · гайд: [production-deploy.md](../production-deploy.md).

**Статус (2026-06-29):** platform stack на `mc.qx-dev.ru` — ✅. Остаётся ops: HTTPS (certbot), бэкапы MySQL.

---

Legend: ☑ = required for MVP alpha · ⊘ = tracked but not blocking MVP · 🤖 = automated unit test in CI

Последнее обновление: 2026-06-29 (prod platform live)
