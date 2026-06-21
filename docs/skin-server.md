# Skin & Cape Server

> **F5:** только **зарегистрированные и авторизованные** пользователи.  
> **Статус:** 🔲 post-MVP · Guest/offline Local — **без** upload и sync (default Steve/Alex).

---

## 1. Architecture

```mermaid
flowchart LR
    User[Registered user] --> Web[/launcher or profile]
    Web --> API[POST /users/me/skin]
    API --> MinIO[(MinIO skins bucket)]
    MC[Minecraft client] --> API[GET /skins/{uuid}.png]
    API --> MinIO
```

Minecraft client configured (via tray launch args) to use QX skin server URL for QX-linked profiles.

---

## 2. Upload rules

| Rule | Value |
| ------ | ------- |
| Format | PNG 64×64 (64×32 legacy supported) |
| Max size | 64 KB |
| Auth | Bearer JWT required |
| Rate limit | 10 uploads / hour / user |
| Storage | `skins/{user_minecraft_uuid}.png` or QX internal UUID |

---

## 3. Cape (post-MVP v1)

- Premium-only or achievement-based — TBD when billing live.
- `GET /capes/{uuid}.png` same pattern.

---

## 4. Launch integration

Tray adds JVM arg for offline/QX profile session:

```text
-Dqx.skinUrl=https://api.qx.example.com/api/v1/skins/{uuid}.png
```

Custom skin mod or authlib injector in classpath for modded — loader-specific (Forge/Fabric post-MVP).

**Vanilla offline:** use established skin override library in QX classpath (implementation detail in
`internal/launcher/skin`).

---

## 5. vs Guest

| Feature | Guest | Registered |
| --------- | ------- | ------------ |
| Upload skin | ✗ | ✓ |
| Public skin URL | ✗ | ✓ |
| Cape | ✗ | post-MVP |

---

*Legal: skins must be user-created or licensed; no Mojang asset rip — [security-legal.md](./security-legal.md)*
