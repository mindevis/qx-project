# Modpacks Pipeline & CurseForge Compliance

> **X4:** CurseForge **primary**, Modrinth fallback.
> Legal: [security-legal.md §5](./security-legal.md)

---

## 1. Import pipeline

```mermaid
flowchart LR
    CF[CurseForge API] --> Importer
    MR[Modrinth API] --> Importer
    Importer --> Norm[QX Manifest Normalizer]
    Norm --> PG[(PostgreSQL)]
    Norm --> MinIO[(Private MinIO)]
    Web[/launcher] --> API
    Tray[QXLauncher] --> API[QXApi]
    Agent[Linux agent] --> API
```

### Search order

1. `GET CurseForge /mods/search`
2. If empty or loader=quilt-only → `GET Modrinth /search`

---

## 2. CurseForge integration

```go
// internal/integrations/curseforge/client.go
// Header: x-api-key: ${CURSEFORGE_API_KEY}
```

| Step | Action |
| ------ | -------- |
| 1 | User picks modpack in `/launcher` |
| 2 | API fetch mod + latest file metadata |
| 3 | Normalize → `QxModpackManifest` + `manifest_sha256` |
| 4 | Download files to MinIO `modpacks/{id}/{sha256}/...` |
| 5 | Store row in `modpacks` |

**Registered users only** for modpack install (B2).

---

## 3. MinIO cache policy

| Object | Visibility | TTL |
| -------- | ------------ | ----- |
| `modpacks/**` | Private | Until project removed from CF |
| Metadata | PG refresh | 24h revalidate |

Presigned URL TTL: **15 minutes** for tray/agent download.

---

## 4. Client ↔ Server sync

Shared `modpack_id`:

1. User assigns modpack to **instance** and **server** in `/launcher`
2. Tray: `modpack.install` on client
3. Agent: `cmd.modpack.install` with same `manifest_sha256`
4. Mismatch → audit alert + UI warning

---

## 5. Mods / shaders / resource packs (registered users)

Separate from full modpacks — per-instance attachments:

| Type | Storage | API |
| ------ | --------- | ----- |
| Mods (.jar) | MinIO `user-content/{user_id}/` | `POST /instances/{id}/mods` |
| Resource packs | same | `POST /instances/{id}/resourcepacks` |
| Shaders | same | `POST /instances/{id}/shaders` |

Guest: **Vanilla only**, no attachments.

Manifest merged at launch time in tray.

---

## 6. Takedown

If CF project deleted:

1. Nightly job marks `modpacks.disabled = true`
2. New installs blocked
3. Existing instances show «modpack unavailable»

---

*Requires registered auth — [security-legal.md §8 RBAC](./security-legal.md)*
