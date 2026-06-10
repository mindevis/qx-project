# QXProject — QA Test Matrix (Alpha)

> Для закрытой beta / MVP Definition of Done.
> Статус теста: ☐ не пройден · ☑ пройден · ⊘ N/A (post-MVP)

---

## 1. Auth & Users

| ID | Сценарий | Шаги | Ожидание | MVP |
| ---- | ---------- | ------ | ---------- | ----- |
| A01 | Register | email + password на panel | 201, login possible | ☑ |
| A02 | Login | valid credentials | JWT, redirect profile | ☑ |
| A03 | Login fail | wrong password | 401 | ☑ |
| A04 | Guest token | POST /auth/guest | guest_token returned | ☑ |
| A05 | Skin upload | auth user POST skin | visible GET /skins/{uuid} | ⊘ |
| A07 | Device register | tray POST register | pending_link | ☑ |
| A08 | Device link | web confirm + tray poll | linked + device_token | ☑ |
| A09 | Tray link menu | ПКМ «Связать» | browser opens /launcher/link | ☑ |

---

## 2. Instances (Client)

| ID | Сценарий | Шаги | Ожидание | MVP |
| ---- | ---------- | ------ | ---------- | ----- |
| I01 | Create instance auth | login + linked device + panel: Vanilla 1.20.4 | instance in list | ☑ |
| I02 | Create instance guest | guest session + web | instance linked to device | ☑ |
| I03 | Launcher sync | open launcher | instance appears | ☑ |
| I04 | Launch Vanilla | Play button | MC client starts | ☑ |
| I05 | Mojang Java | fresh install | Mojang JRE downloaded | ☑ |
| I06 | Modpack instance | select modpack | ⊘ post-MVP | ⊘ |
| I07 | Forge/NeoForge launch | modded client | ⊘ post-MVP | ⊘ |

---

## 3. Launcher UI (React on website)

| ID | Сценарий | MVP |
| ---- | ---------- | ----- |
| L01 | /launcher page loads | ☑ |
| L02 | Device link flow | ☑ |
| L03 | Tray «Связать лаунчер» | ☑ |
| L04 | Public servers tab | ⊘ |

---

## 4. Servers & Agent (Linux)

| ID | Сценарий | Шаги | Ожидание | MVP |
| ---- | ---------- | ------ | ---------- | ----- |
| S01 | Create server | panel + SSH creds | server pending | ☑ |
| S02 | SSH deploy | POST deploy | agent online, systemd running | ☑ |
| S03 | Start server | Paper/Vanilla jar | status running | ☑ |
| S04 | Stop server | stop button | process killed | ☑ |
| S05 | Live console | WS console | stdout visible | ☑ |
| S06 | Console input | type command | executed in MC | ☑ |
| S07 | Multi-admin invite | add admin user | admin can start | ⊘ |
| S08 | Viewer read-only | viewer login | console read, no start | ⊘ |
| S09 | Modpack server sync | same modpack_id | agent installs to correct dirs by loader | ⊘ |
| S10 | Hybrid jar (Mohist) | server_type hybrid + mohist | starts; mods/ + plugins/ | ⊘ |
| S10a | Plugins on Paper | POST /servers/{id}/plugins | installed to plugins/ | ⊘ |
| S10b | Mods on NeoForge | POST /servers/{id}/mods | installed to mods/ | ⊘ |
| S10c | Reject mods on Paper | POST /servers/{id}/mods | 403 CONTENT_NOT_ALLOWED | ⊘ |
| S10d | Reject plugins on NeoForge | POST /servers/{id}/plugins | 403 CONTENT_NOT_ALLOWED | ⊘ |
| S10e | Hybrid both content types | mods + plugins on Mohist | both dirs populated | ⊘ |
| S11 | Agent reconnect | restart API | agent reconnects ≤60s | ☑ |
| S12 | Idempotent start | duplicate request_id | no double process | ☑ |

---

## 5. Modpack sync (post-MVP)

| ID | Сценарий | Ожидание | MVP |
| ---- | ---------- | ---------- | ----- |
| M01 | Same modpack_id client+server | hash match both sides | ⊘ |
| M02 | CurseForge import | manifest in DB | ⊘ |
| M03 | Modrinth mrpack | manifest in DB | ⊘ |

---

## 6. Infra & Non-functional

| ID | Сценарий | Ожидание | MVP |
| ---- | ---------- | ---------- | ----- |
| N01 | Docker Compose prod | all services up | ☑ |
| N02 | TLS | HTTPS valid | ☑ |
| N03 | MySQL backup restore | data intact | ⊘ |
| N04 | Agent non-Linux | deploy to Windows — fail gracefully (Linux only) | ☑ |

---

## 7. Regression checklist (pre-release)

- [ ] All ☑ MVP rows passed
- [ ] No P0/P1 open bugs
- [ ] `docs/api.md` matches implemented routes
- [ ] `agent-protocol.md` message types match code
- [ ] Junior completed manual pass with notes

---

Legend: ☑ = required for MVP alpha · ⊘ = tracked but not blocking MVP
