# Modpacks Pipeline & Client Content Install

> **Статус реализации:** 🔲 post-MVP (v2+). REST paths — относительно `/api/v1`.
> **X4:** CurseForge **primary**, Modrinth fallback.  
> **ADR-0011:** файлы инстанса — **на ПК через QXLauncher**, не в MinIO.  
> Legal: [security-legal.md §5](./security-legal.md)

---

## 1. Принцип

| Где | Что |
| ----- | ----- |
| **QXApi / MySQL** | Метаданные modpack, `QxModpackManifest`, `manifest_sha256`, ссылки на файлы |
| **ПК пользователя** | Mods, modpacks, shaders, resource packs — **файлы в папке инстанса** |
| **QXLauncher** | Скачивание, verify hash, assemble, локальный cache |
| **MinIO** | **Не** для client/server mod content — только platform blobs (см. [architecture.md §4](./architecture.md), [ADR-0011](./adr/0011-client-local-content-install.md)) |

```mermaid
flowchart LR
    CF[CurseForge API] --> QXApi
    MR[Modrinth API] --> QXApi
    Mojang[Mojang assets] --> QXLauncher
    QXWeb[/launcher] --> QXApi
    QXApi --> MySQL[(MySQL manifest)]
    QXLauncher --> QXApi
    QXLauncher --> CF
    QXLauncher --> MR
    QXLauncher --> Disk[Instance dir on PC]
```

---

## 2. Import / catalog (QXApi)

```go
// internal/integrations/curseforge/client.go
// Header: x-api-key: ${CURSEFORGE_API_KEY}
```

| Step | Action |
| ------ | -------- |
| 1 | User picks modpack in QXWeb `/launcher` |
| 2 | QXApi fetch mod + file metadata from CF/MR |
| 3 | Normalize → `QxModpackManifest` + `manifest_sha256` |
| 4 | Save row in `modpacks` (**MySQL only**, no file upload) |

**Registered users only** for modpack install (B2).

### Search order

1. `GET CurseForge /mods/search`
2. If empty or loader=quilt-only → `GET Modrinth /search`

---

## 3. Install on PC (QXLauncher)

| Step | Action |
| ------ | -------- |
| 1 | `GET /instances/{id}/manifest` or `GET /modpacks/{id}/manifest` |
| 2 | QXLauncher resolves download URLs (CF `download-url`, MR CDN, Mojang) |
| 3 | Download to `{instance_root}/mods/`, `resourcepacks/`, `shaderpacks/` |
| 4 | Verify SHA256 / MD5 from manifest |
| 5 | Local cache: `%AppData%/QX/cache/` (or `~/.local/share/qx/cache`) — delta on re-install |

Guest: **Vanilla only**, no mods/shaders/resource packs.

---

## 4. Mods / shaders / resource packs (registered)

Per-instance attachments — metadata в MySQL (`launcher_instances.mods`, `resource_packs`, `shaders` JSON),
**файлы на диске ПК**:

| Type | On disk | API |
| ------ | --------- | ----- |
| Mods (.jar) | `{instance}/mods/` | `POST /instances/{id}/mods` → manifest entry |
| Resource packs | `{instance}/resourcepacks/` | `POST /instances/{id}/resourcepacks` |
| Shaders | `{instance}/shaderpacks/` | `POST /instances/{id}/shaders` |

QXLauncher materializes files at sync / pre-launch; QXApi не хранит `.jar` / `.zip`.

---

## 5. Client ↔ Server sync (BYOS)

Shared `modpack_id` + `manifest_sha256`:

1. User assigns modpack to **instance** and **server** in `/launcher`
2. **QXLauncher:** install to PC instance dir (client loader)
3. **QXAgent:** install to server disk — **mods/** and/or **plugins/** per [server-content-install.md](./server-content-install.md)
4. QXApi rejects incompatible pairs (e.g. Fabric modpack + Paper server) before deploy
5. Mismatch hash → audit alert + UI warning

---

## 6. Takedown

If CF project deleted:

1. Nightly job marks `modpacks.disabled = true`
2. New installs blocked
3. Existing instances show «modpack unavailable»

---

*Requires registered auth — [security-legal.md §8 RBAC](./security-legal.md)*
