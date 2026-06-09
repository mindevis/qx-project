# Mojang Java Runtime

> **F1:** предпочтительно **Mojang Java** (official runtime), не Adoptium, для client launch.

---

## 1. Strategy

| Priority | Source |
| ---------- | -------- |
| 1 | Mojang-provided Java manifest (launcher metadata) |
| 2 | Cached copy in `%LOCALAPPDATA%/QX/java/` (Win) |
| 3 | Fallback: system `JAVA_HOME` (warn user, post-MVP) |

Tray downloads and verifies Mojang runtime per MC version family.

---

## 2. Version matrix

| Minecraft | Java | Mojang component | Notes |
| ----------- | ------ | ------------------ | ------- |
| 1.16.x | **Java 8** | jre-legacy | Last 8 line |
| 1.17.x | **Java 16** | java-runtime-alpha | Transitional |
| 1.18.x – 1.20.4 | **Java 17** | java-runtime-gamma | LTS |
| 1.20.5+ | **Java 21** | java-runtime-delta | Current |
| 1.21.x | **Java 21** | java-runtime-delta | Verify manifest |

**Server-side (agent):** same matrix; agent installs `headless` JRE on Linux VPS if missing during deploy.

---

## 3. Download flow (tray)

```mermaid
flowchart TD
    A[Resolve MC version] --> B{Java cached?}
    B -->|yes| C[Verify SHA256]
    B -->|no| D[GET Mojang java-component manifest]
    D --> E[Download platform archive]
    E --> F[Extract to QX/java/{component}/{version}/]
    F --> C
    C --> G[Use java binary in launch]
```

Platform keys: `linux`, `windows-x64`, `mac-os-x64`, `mac-os-arm64`.

---

## 4. Paths

| OS | Base path |
| ---- | ----------- |
| Windows | `%LOCALAPPDATA%\QX\java\` |
| macOS | `~/Library/Application Support/QX/java/` |
| Linux | `~/.local/share/qx/java/` |

Structure:

```text
java/
  java-runtime-delta/
    21.0.3/
      bin/java
      ...
```

---

## 5. Server agent (Linux)

During SSH deploy optional step:

```bash
/opt/qx/java/bin/java -version || qx-agent install-java --component java-runtime-delta
```

Agent uses `/opt/qx/java/bin/java` in systemd `ExecStart` wrapper for MC server.

---

## 6. Manifest integration

From MC `version.json`:

```json
{
  "javaVersion": {
    "component": "java-runtime-delta",
    "majorVersion": 21
  }
}
```

Tray maps `component` → download URL via Mojang meta API (same as official launcher).

---

## 7. Disk budget

| Component | ~Size |
| ----------- | ------- |
| Java 8 legacy | 80 MB |
| Java 17 | 180 MB |
| Java 21 | 190 MB |

UI shows «Installed runtimes» in `/launcher/settings/java` with delete option.

---

## 8. Errors

| Code | User message |
| ------ | -------------- |
| `JAVA_DOWNLOAD_FAILED` | Retry; check internet |
| `JAVA_VERIFY_FAILED` | Re-download |
| `JAVA_UNSUPPORTED_OS` | Update OS / tray |

---

*См. [launch-bridge.md](./launch-bridge.md)*
