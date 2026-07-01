# QX Launcher and Windows Defender

> Why `Program:Win32/Wacapew.A!ml` (or similar) may appear, what we changed to reduce false positives, and how to report a misdetection to Microsoft.

---

## 1. Why Defender may flag QX Launcher

`Wacapew.A!ml` is a **machine-learning** detection name (`!ml` = ML model). It is not a verdict on a specific known malware family. Defender scores **behavior and reputation**, and unsigned tray apps that download and replace executables often score high.

Common triggers in QX Launcher (pre-mitigation):

| Signal | Where | Why AV cares |
| ------ | ----- | ------------ |
| **Self-update** | `internal/updater` | Download EXE → replace running process → restart |
| **Helper batch script** | *(removed)* `qxlauncher-update.cmd` via `cmd /C` | Classic dropper / updater pattern |
| **Hidden subprocesses** | `internal/proc` (`CREATE_NO_WINDOW`) | Concealed child processes |
| **Living-off-the-land** | *(removed for URLs)* `rundll32 url.dll,FileProtocolHandler` | LOLBin abuse pattern |
| **PowerShell toasts** | `internal/notify` | Script execution from a GUI binary |
| **No console** | `-H=windowsgui` | GUI-only binary with network + process spawn |
| **Unsigned binary** | release build | No Authenticode reputation |

None of these are malicious by themselves; together they resemble generic “downloader / dropper” heuristics.

---

## 2. Mitigations in this repo

| Change | Effect |
| ------ | ------ |
| **In-place rename update** | Running `qx-launcher.exe` is renamed to `.prev`, new bytes are copied to the original path, new process starts — **no `.cmd` / `cmd.exe` helper** |
| **Staging under `%LOCALAPPDATA%\QXLauncher\updates`** | Download does not sit beside the installed EXE with a suspicious temp name |
| **PE version resource + manifest** | `goversioninfo` embeds company/product/description and `asInvoker` manifest at build (`make build-launcher-win`) |
| **ShellExecute for URLs** | `browser.Open` uses `shell32.dll` instead of `rundll32` |
| **Startup cleanup** | Removes `qx-launcher.exe.prev` after a successful update |

**Still present (lower risk, harder to replace):**

- PowerShell for Windows toast notifications (`internal/notify`) — native WinRT toasts are planned; systray already covers most UX.
- `CREATE_NO_WINDOW` for Java/Minecraft and other child processes — required so game launches do not flash console windows.
- Unsigned Authenticode — **the main long-term fix** (see below).

---

## 3. Code signing (recommended for production)

Authenticode signing is the most effective way to build SmartScreen/Defender reputation.

### 3.1 Obtain a certificate

- Purchase an **OV or EV** code-signing certificate from a public CA (DigiCert, Sectigo, etc.).
- EV certificates enable faster Microsoft reputation; OV is acceptable for smaller releases.

### 3.2 Sign the release binary

After `make build-launcher-win`:

```powershell
# signtool is part of the Windows SDK
signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 `
  /a bin\qx-launcher.exe
```

Use your publisher name consistently; it should match the PE `CompanyName` (`QX Project`).

### 3.3 CI integration (optional)

Store the `.pfx` in GitHub Actions secrets and add a signing step to `.github/workflows/prod-release.yml` **before** the binary is copied into the web image. Never commit the private key.

---

## 4. Report a false positive to Microsoft

If Defender still blocks a **signed** or **latest** build:

1. **Windows Security → Protection history** — note detection name, file path, and hash.
2. Submit the file: [Microsoft Security Intelligence — Submit a file](https://www.microsoft.com/en-us/wdsi/filesubmission) — choose **Software developer** / false positive.
3. Include:
   - Product: **QX Launcher**
   - Publisher: **QX Project**
   - Download URL: your official `qx-launcher.exe` link (e.g. `https://mc.qx-dev.ru/downloads/qx-launcher.exe`)
   - SHA-256 of the exact binary
   - Short explanation: legitimate Minecraft launcher; self-update uses in-place rename (no batch); no persistence beyond user startup
4. For enterprise: [Submit via MDE portal](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/defender-endpoint-false-positives-negatives) if applicable.

Microsoft typically re-evaluates within 1–3 business days.

### 4.1 User workaround (until reputation improves)

- **Windows Security → Virus & threat protection → Manage settings → Exclusions** — add the folder containing `qx-launcher.exe` (user choice; document in FAQ).
- Or: Protection history → **Allow** on the specific detection (per-event).

---

## 5. Verify PE metadata locally

After building:

```powershell
(Get-Item bin\qx-launcher.exe).VersionInfo | Format-List *
```

Expect `CompanyName`, `ProductName`, `FileDescription`, and `OriginalFilename` populated.

---

*See also [auto-update.md](./auto-update.md) (signing roadmap), [security-legal.md](./security-legal.md) §10.*

Last updated: 2026-07-01
