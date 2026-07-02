# QX Cosmetics

QX Cosmetics is QXSystem's account-wide **skin service**. Users upload a custom skin and choose Steve or Alex model on the **Profile** page. QXLauncher applies the skin at game launch — **no client mods required**.

## What works today

| Feature | Offline profile | Mojang-linked |
|---------|----------------|---------------|
| Custom skin upload + preview (web) | Yes | Yes |
| Skin served at public URL | Yes | Yes |
| Skin in-game (vanilla client) | Yes* | Yes* |

\*QXLauncher enables the **QX Skin Server** (Ely.by-style session host override) when a custom skin is equipped. The client fetches a session profile with your skin texture URL — no Fabric mod or authlib-injector JAR needed. A fallback copy is also written to `gameDir/skins/{uuid}.png`.

For Mojang-linked accounts, upload a custom skin on Profile to override the official Mojang skin for QX launches.

### Out of scope (requires client mods)

Capes, wings, and other non-vanilla cosmetics are **not supported** (would require client mods; QX uses the skin server approach instead).

## Flow

1. User uploads skin and selects Steve/Alex model on **Profile → Skin**.
2. QXApi stores metadata in `user_cosmetics` and the PNG under `cosmetics_data_dir`.
3. On launch, QXApi includes skin metadata in the pending launch payload (`skin_model`, `skin_url`, `use_skin_server`, `skin_server_host`, `game_uuid`).
4. QXLauncher downloads the skin texture, writes `gameDir/skins/{uuid}.png`, and prepends skin-server JVM args when `use_skin_server` is set.

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users/me/cosmetics` | Bearer | Current skin settings |
| PUT | `/api/v1/users/me/cosmetics` | Bearer | Set `{ skin_model }` |
| POST | `/api/v1/users/me/cosmetics/skin` | Bearer | Multipart field `skin` (PNG 64×64 or 64×32) |
| DELETE | `/api/v1/users/me/cosmetics/skin` | Bearer | Remove custom skin |
| GET | `/api/v1/cosmetics/skins/{userId}.png` | Public | Skin texture (by game UUID or QX user id) |
| GET | `/sessionserver/session/minecraft/profile/{uuid}` | Public | Yggdrasil-compatible profile with skin textures |

## Configuration (`qxapi.toml`)

```toml
# Directory for uploaded skin PNG files (default: data/cosmetics)
cosmetics_data_dir = "data/cosmetics"
```

Environment override: `COSMETICS_DATA_DIR`.

`public_api_url` must be reachable by QXLauncher and the game client for skin URLs and the skin server session host.

## Web preview

The profile skin preview uses [skinview3d](https://www.npmjs.com/package/skinview3d) (MIT).

## Database notes

Legacy `cape_type`, `wings_type`, and `has_cape` columns remain in `user_cosmetics` for backward compatibility but are not exposed via API or UI.
