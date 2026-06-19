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
| E2E API smoke | Flow A/B/C router | `make e2e-api-smoke` | 🤖 ☑ |
| E2E launch-bridge dry-run | tray poll → dry JVM | `make e2e-dry-run` | 🤖 ☑ |
| E2E JVM smoke | Mojang manifest + `java -version` | `make e2e-jvm` | 🤖 ☑ partial (I04/I05) |
| E2E (Playwright / manual tray+JVM) | A09, L03, I04 full MC client | `make e2e-alpha` + `make e2e-manual` | 🤖 ☑ Flow A+B+C web + API + dry-run · ☐ tray/full JVM |

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
| A07 | Device register | tray POST register | pending_link | ☑ | 🤖 ☑ `router_test` |
| A08 | Device link | web confirm + tray poll | linked + device_token | ☑ | 🤖 ☑ Flow A/B |
| A09 | Tray link menu | ПКМ «Связать» | browser opens /launcher/link | ☑ | ☐ manual tray |

---

## 2. Instances (Client)

| ID | Сценарий | Шаги | Ожидание | MVP | Phase 0 |
| ---- | ---------- | ------ | ---------- | ----- | --------- |
| I01 | Create instance auth | login + linked device + panel: Vanilla 1.20.4 | instance in list | ☑ | 🤖 ☑ Flow A |
| I02 | Create instance guest | guest session + web | instance linked to device | ☑ | 🤖 ☑ Flow B |
| I03 | Launcher sync | open launcher | instance appears | ☑ | 🤖 ☑ tray `syncInstances` |
| I04 | Launch Vanilla | Play button | MC client starts | ☑ | 🤖 API + dry-run ☑ · `make e2e-jvm` ☑ · ☐ manual full MC |
| I05 | Mojang Java | fresh install | Mojang JRE downloaded | ☑ | 🤖 `make e2e-jvm` (PATH java) ☑ · ☐ manual Mojang JRE download |
| I06 | Modpack instance | select modpack | ⊘ post-MVP | ⊘ | ⊘ |
| I07 | Forge/NeoForge launch | modded client | ⊘ post-MVP | ⊘ | ⊘ |

---

## 3. Launcher UI (React on website)

| ID | Сценарий | MVP | Phase 0 |
| ---- | ---------- | ----- | --------- |
| L01 | /launcher page loads (placeholder) | ☑ | ☑ instances + profiles UI |
| L02 | Device link flow | ☑ | 🤖 ☑ `LauncherLinkPage` tests |
| L03 | Tray «Связать лаунчер» | ☑ | ☐ manual systray |
| L04 | Public servers tab | ⊘ | ⊘ |

---

## 4. Servers & Agent (Linux)

| ID | Сценарий | Шаги | Ожидание | MVP | Phase 0 |
| ---- | ---------- | ------ | ---------- | ----- | --------- |
| S01 | Create server | panel + SSH creds | server pending | ☑ | 🤖 ☑ `ServersPage` tests |
| S02 | SSH deploy | POST deploy | agent online, systemd running | ☑ | 🤖 ☑ |
| S03 | Start server | Paper/Vanilla jar | status running | ☑ | 🤖 ☑ |
| S04 | Stop server | stop button | process killed | ☑ | 🤖 ☑ |
| S05 | Live console | WS console | stdout visible | ☑ | 🤖 ☑ |
| S06 | Console input | type command | executed in MC | ☑ | 🤖 ☑ |
| S07 | Multi-admin invite | add admin user | admin can start | ⊘ | ⊘ |
| S08 | Viewer read-only | viewer login | console read, no start | ⊘ | ⊘ |
| S09 | Modpack server sync | same modpack_id | agent installs by loader | ⊘ | ⊘ |
| S10 | Hybrid jar (Mohist) | server_type hybrid | mods/ + plugins/ | ⊘ | ⊘ |
| S10a | Plugins on Paper | POST /servers/{id}/plugins | plugins/ | ⊘ | ⊘ |
| S10b | Mods on NeoForge | POST /servers/{id}/mods | mods/ | ⊘ | ⊘ |
| S10c | Reject mods on Paper | POST /servers/{id}/mods | 403 CONTENT_NOT_ALLOWED | ⊘ | ⊘ |
| S10d | Reject plugins on NeoForge | POST /servers/{id}/plugins | 403 | ⊘ | ⊘ |
| S10e | Hybrid both content types | mods + plugins Mohist | both dirs | ⊘ | ⊘ |
| S11 | Agent reconnect | restart API | agent reconnects ≤60s | ☑ | 🤖 ☑ `agenthub` reconnect test |
| S12 | Idempotent start | duplicate request_id | no double process | ☑ | 🤖 ☑ agent `requestCache` replay |

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
| N04 | Agent non-Linux | deploy to Windows SSH — fail gracefully | ☑ | 🤖 ☑ `ErrNonLinuxHost` |

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
