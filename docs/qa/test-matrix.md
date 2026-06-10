# QXProject — QA Test Matrix (Alpha)

> REST base: `http://localhost:3000/api/v1` (prod: `https://api.qx.example.com/api/v1`).  
> Пути в таблицах — относительные к base, если не указано иное.
>
> Для закрытой beta / MVP Definition of Done.
> Статус теста: ☐ не пройден · ☑ пройден · ⊘ N/A (post-MVP) · 🤖 автоматизирован (unit)
> Server content (mods/plugins): [server-content-install.md](../server-content-install.md)

---

## 0. Automated unit tests (CI)

| Область | Покрытие | Команда | Статус |
| --------- | ---------- | --------- | -------- |
| `services/qxapi` | 100% Go statements | `cd services/qxapi && go test ./...` | 🤖 ☑ |
| `services/qxagent/cmd` | 100% (stub) | `cd services/qxagent && go test ./...` | 🤖 ☑ |
| `services/qxlauncher/cmd` | 100% (stub) | `cd services/qxlauncher && go test ./...` | 🤖 ☑ |
| `web/qxweb` | 100% stmts/branches | `cd web/qxweb && npm run test:coverage` | 🤖 ☑ |
| E2E (Playwright / manual) | — | Phase Alpha | ☐ |

---

## 1. Auth & Users

| ID | Сценарий | Шаги | Ожидание | MVP | Phase 0 |
| ---- | ---------- | ------ | ---------- | ----- | --------- |
| A01 | Register | email + password на panel | 201, login possible | ☑ | 🤖 ☑ |
| A02 | Login | valid credentials | JWT, redirect `/profile` | ☑ | 🤖 ☑ |
| A02b | Change password | PATCH `/users/me/password` | 204, login with new password | ☑ | 🤖 ☑ |
| A02c | Change email | PATCH `/users/me/email` | 200, updated profile | ☑ | 🤖 ☑ |
| A03 | Login fail | wrong password | 401 | ☑ | 🤖 ☑ |
| A04 | Guest token | POST /auth/guest | guest_token returned | ☑ | 🤖 ☑ |
| A05 | Skin upload | auth user POST skin | visible GET /skins/{uuid} | ⊘ | ☐ |
| A07 | Device register | tray POST register | pending_link | ☑ | ☐ Phase 1 |
| A08 | Device link | web confirm + tray poll | linked + device_token | ☑ | ☐ Phase 1 |
| A09 | Tray link menu | ПКМ «Связать» | browser opens /launcher/link | ☑ | ☐ Phase 1 |

---

## 2. Instances (Client)

| ID | Сценарий | Шаги | Ожидание | MVP | Phase 0 |
| ---- | ---------- | ------ | ---------- | ----- | --------- |
| I01 | Create instance auth | login + linked device + panel: Vanilla 1.20.4 | instance in list | ☑ | ☐ Phase 1 |
| I02 | Create instance guest | guest session + web | instance linked to device | ☑ | ☐ Phase 1 |
| I03 | Launcher sync | open launcher | instance appears | ☑ | ☐ Phase 1 |
| I04 | Launch Vanilla | Play button | MC client starts | ☑ | ☐ Phase 1 |
| I05 | Mojang Java | fresh install | Mojang JRE downloaded | ☑ | ☐ Phase 1 |
| I06 | Modpack instance | select modpack | ⊘ post-MVP | ⊘ | ⊘ |
| I07 | Forge/NeoForge launch | modded client | ⊘ post-MVP | ⊘ | ⊘ |

---

## 3. Launcher UI (React on website)

| ID | Сценарий | MVP | Phase 0 |
| ---- | ---------- | ----- | --------- |
| L01 | /launcher page loads (placeholder) | ☑ | ☑ placeholder only |
| L02 | Device link flow | ☑ | ☐ Phase 1 |
| L03 | Tray «Связать лаунчер» | ☑ | ☐ Phase 1 |
| L04 | Public servers tab | ⊘ | ⊘ |

---

## 4. Servers & Agent (Linux)

| ID | Сценарий | Шаги | Ожидание | MVP | Phase 0 |
| ---- | ---------- | ------ | ---------- | ----- | --------- |
| S01 | Create server | panel + SSH creds | server pending | ☑ | ☐ Phase 2 |
| S02 | SSH deploy | POST deploy | agent online, systemd running | ☑ | ☐ Phase 2 |
| S03 | Start server | Paper/Vanilla jar | status running | ☑ | ☐ Phase 2 |
| S04 | Stop server | stop button | process killed | ☑ | ☐ Phase 2 |
| S05 | Live console | WS console | stdout visible | ☑ | ☐ Phase 2 |
| S06 | Console input | type command | executed in MC | ☑ | ☐ Phase 2 |
| S07 | Multi-admin invite | add admin user | admin can start | ⊘ | ⊘ |
| S08 | Viewer read-only | viewer login | console read, no start | ⊘ | ⊘ |
| S09 | Modpack server sync | same modpack_id | agent installs by loader | ⊘ | ⊘ |
| S10 | Hybrid jar (Mohist) | server_type hybrid | mods/ + plugins/ | ⊘ | ⊘ |
| S10a | Plugins on Paper | POST /servers/{id}/plugins | plugins/ | ⊘ | ⊘ |
| S10b | Mods on NeoForge | POST /servers/{id}/mods | mods/ | ⊘ | ⊘ |
| S10c | Reject mods on Paper | POST /servers/{id}/mods | 403 CONTENT_NOT_ALLOWED | ⊘ | ⊘ |
| S10d | Reject plugins on NeoForge | POST /servers/{id}/plugins | 403 | ⊘ | ⊘ |
| S10e | Hybrid both content types | mods + plugins Mohist | both dirs | ⊘ | ⊘ |
| S11 | Agent reconnect | restart API | agent reconnects ≤60s | ☑ | ☐ Phase 2 |
| S12 | Idempotent start | duplicate request_id | no double process | ☑ | ☐ Phase 2 |

---

## 5. Modpack sync (post-MVP)

| ID | Сценарий | Ожидание | MVP |
| ---- | ---------- | ---------- | ----- |
| M01 | Same modpack_id client+server | hash match both sides | ⊘ |
| M02 | CurseForge import | manifest in DB | ⊘ |
| M03 | Modrinth mrpack | manifest in DB | ⊘ |

---

## 6. Infra & Non-functional

| ID | Сценарий | Ожидание | MVP | Phase 0 |
| ---- | ---------- | ---------- | ----- | --------- |
| N01 | Docker Compose dev | MySQL, Redis, MinIO up | ☑ | ☑ `make dev-up` |
| N02 | TLS | HTTPS valid | ☑ | ☐ prod |
| N03 | MySQL backup restore | data intact | ⊘ | ⊘ |
| N04 | Agent non-Linux | deploy to Windows — fail gracefully | ☑ | ☐ Phase 2 |

---

## 7. Regression checklist (pre-release)

- [ ] All ☑ MVP rows passed (manual + E2E)
- [ ] All 🤖 unit tests green in CI
- [ ] No P0/P1 open bugs
- [ ] `docs/api.md` matches implemented routes
- [ ] `agent-protocol.md` message types match code (Phase 2+)
- [ ] Junior completed manual pass with notes

---

Legend: ☑ = required for MVP alpha · ⊘ = tracked but not blocking MVP · 🤖 = automated unit test in CI

Последнее обновление: 2026-06-10 (v1.6 — REST `/api/v1`)
