# Device Linking — Связь Launcher ↔ Сайт

> **Статус реализации:** ✅ Phase 1+ — `POST /api/v1/launcher/devices/*`, web `/launcher/link`, tray poll; manual ☑ (A09, L03).
> REST base: `/api/v1` — пути в §3 API относительные к base.
> Решение **E6**: обязательная привязка лаунчера к сайту **до** создания инстансов и игры.
> Работает **без регистрации** (guest) и **с аккаунтом** (после login — re-link или merge).

---

## 1. Принцип

| Правило | Описание |
| --------- | ---------- |
| **Обязательно** | Нельзя создавать инстансы и играть, пока `launcher_devices.status ≠ linked` |
| **Registered** | Login на сайте → confirm link → `device_token` привязан к `user_id` |
| **Без регистрации** | Guest на сайте + linked device = полный guest-flow |
| **UI на сайте** | Управление инстансами — **`/launcher`** (React SPA), не в окне tray |
| **Go daemon** | Tray: launch-bridge poll, JVM, Java, уведомления |
| **Guest vs Registered** | Guest (linked): Vanilla, Local. Registered+auth: mods, shaders, resource packs, modpacks — [security-legal §8](./security-legal.md) |

---

## 2. User Flow

```mermaid
sequenceDiagram
    participant U as Пользователь
    participant Web as Сайт /launcher
    participant API as Backend
    participant L as QXLauncher (tray)

    U->>Web: Раздел «Лаунчер» → Скачать
    U->>L: Запуск qx-launcher
    L->>API: POST /api/v1/launcher/devices/register
    API-->>L: pending_link, user_code, link_url
    L->>U: OS notification «Свяжите лаунчер с сайтом»
    U->>Web: Открыть link_url (или /launcher/link)
    U->>Web: Подтвердить привязку (guest cookie или login)
    Web->>API: POST /api/v1/launcher/devices/link
    API-->>L: poll: status=linked, device_token
    L->>U: Notification «Лаунчер связан»
    U->>Web: Создать инстанс
    L->>API: GET /api/v1/instances (device_token)
    L->>L: Install + Launch
```

### 2.1 Шаги для пользователя

1. Зайти на сайт → **Лаунчер** → скачать установщик.
2. Запустить **qx-launcher** (иконка в трее).
3. Увидеть **уведомление ОС**: «Laунcher нужно связать с сайтом».
4. Связать одним из способов:
   - Клик по уведомлению → браузер → страница подтверждения;
   - **ПКМ по иконке в трее** → **«Связать QXLauncher»** → браузер;
   - Вручную: сайт → `/launcher/link` → код с tray (если показан).
5. На сайте нажать **«Подтвердить»** (guest-сессия создаётся автоматически, если не залогинен).
6. Создать инстанс на сайте → лаунчер sync → **Играть**.

---

## 3. Tray UI (Go launcher)

| Элемент | Поведение |
| --------- | ----------- |
| Иконка в трее | Статус: 🔴 не связан / 🟢 связан |
| ЛКМ | Открыть сайт `/launcher` в браузере *(Windows: через пункт меню «Открыть сайт» — ограничение `fyne.io/systray`)* |
| ПКМ меню | «Связать QXLauncher» (если не linked) · «Открыть сайт» · «Выход» |
| Уведомления | pending_link · linked · update available · instance ready |

**WebView не используется** — весь UI на сайте ([ADR-0006](./adr/0006-launcher-website-ui.md)).

---

## 4. API

| Method | Path | Кто | Описание |
| -------- | ------ | ----- | ---------- |
| POST | `/launcher/devices/register` | Go app | `{ device_id, os, hostname, launcher_version }` |
| GET | `/launcher/devices/{id}/status` | Go app | Poll: `pending_link` or `linked` |
| POST | `/launcher/devices/link` | Web SPA | `{ device_id, user_code? }` + guest cookie или JWT |
| POST | `/launcher/devices/unlink` | Web / Go | Отвязка |
| GET | `/launcher/devices/pending` | Web SPA | Список ожидающих (same browser IP/session) — optional |

### Register response (pending)

```json
{
  "device_id": "uuid",
  "status": "pending_link",
  "user_code": "ABCD-1234",
  "link_url": "https://qx.example.com/launcher/link?device=uuid",
  "poll_interval_sec": 3,
  "expires_at": "2026-06-09T13:00:00Z"
}
```

### Link response (Go poll)

```json
{
  "status": "linked",
  "device_token": "eyJ...",
  "owner_type": "guest",
  "guest_session_id": "uuid"
}
```

После link Go сохраняет `device_token` локально (`~/.config/qx/device.json`).

---

## 5. Guest → User migration (при регистрации)

Когда пользователь **регистрируется** или **логинится** на сайте, уже имея linked guest device:

```mermaid
sequenceDiagram
    participant Web as Сайт
    participant API as Backend
    participant L as Launcher

    Note over L: device linked to guest_session G1
    U->>Web: Register / Login
    Web->>API: POST /api/v1/auth/register { ..., device_id }
    API->>API: Merge G1 → user: instances, profiles
    API-->>L: new device_token (owner=user)
```

| Данные | Политика merge |
| -------- | ---------------- |
| `launcher_instances` | `guest_session_id` → `user_id` |
| `offline_profiles` | Перенос на `user_id` |
| `launcher_devices` | `guest_session_id` → `user_id` |

Конфликт имён инстансов: suffix `(imported)`.

---

## 6. Security

| Риск | Митигация |
| ------ | ----------- |
| Hijack device link | `user_code` + короткий TTL (15 min) + confirm на сайте |
| Stolen device_token | Rotate on re-link; bind to device_id |
| Fake register | Rate limit по IP; device_id UUID v4 local storage |

---

## 7. State machine

```mermaid
stateDiagram-v2
    [*] --> Unregistered: first launch
    Unregistered --> PendingLink: POST register
    PendingLink --> Linked: POST link (web)
    PendingLink --> Expired: TTL 15m
    Expired --> PendingLink: re-register
    Linked --> PendingLink: unlink
    Linked --> Linked: guest→user merge
```

---

*См. [api.md](./api.md), [schema.sql](./schema.sql), [launch-bridge.md](./launch-bridge.md)*

Последнее обновление: 2026-06-10
