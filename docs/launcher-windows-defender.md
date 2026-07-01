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
| **PowerShell toasts** | *(removed)* `internal/notify` | Script execution from a GUI binary |
| **No console** | `-H=windowsgui` | GUI-only binary with network + process spawn |
| **Unsigned binary** | release build | No Authenticode reputation |

None of these are malicious by themselves; together they resemble generic “downloader / dropper” heuristics.

---

## 2. Mitigations in this repo

| Change | Effect |
| ------ | ------ |
| **In-place rename update** | Running `qx-launcher.exe` is renamed to `.prev`, new bytes are copied to the original path, new process starts — **no `.cmd` / `cmd.exe` helper** |
| **Staging under `%LOCALAPPDATA%\QXLauncher\updates`** | Download does not sit beside the installed EXE with a suspicious temp name |
| **PE validation before install** | Staging must be ≥64 KiB and start with `MZ` before replacing the running binary |
| **PE version resource + manifest** | `goversioninfo` embeds company/product/description and `asInvoker` manifest at build (`make build-launcher-win`) |
| **ShellExecute for URLs** | `browser.Open` uses `shell32.dll` instead of `rundll32` |
| **WinRT toast notifications** | `go-toast` COM API — **no PowerShell** from QX Launcher for desktop toasts |
| **`-trimpath` release builds** | Strips host build paths from the binary (reproducible, less “one-off” heuristic noise) |
| **Startup cleanup** | Removes `qx-launcher.exe.prev` after a successful update |

**Still present (lower risk, harder to replace without product changes):**

- **Self-update** (download + in-place replace + `os.Exit`) — required for tray auto-update today; long-term alternative is **MSI/WiX** or store distribution (see §6).
- `CREATE_NO_WINDOW` for Java/Minecraft child processes — required so game launches do not flash console windows.
- **Unsigned Authenticode** — **the main long-term fix** (see §3).

### 2.1 CI / build flags

`make build-launcher-win` uses:

```text
go build -trimpath -ldflags "-H=windowsgui \
  -X .../config.embeddedAPIBaseURL=... \
  -X .../config.embeddedWebBaseURL=... \
  -X .../version.Version=..."
```

There are **no suspicious ldflags** (no `-s -w` stripping, no exotic `-buildmode`). `-H=windowsgui` is standard for systray apps. Embedded URLs are the production API/web bases, not obfuscation.

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

### 3.3 Self-signed certificate (development only)

For **local dev** or internal testers, a self-signed cert reduces some client-side warnings but **does not** build Microsoft cloud reputation. Useful to verify signing plumbing before buying a CA cert:

```powershell
# Create dev cert (once)
$cert = New-SelfSignedCertificate -Type CodeSigningCert `
  -Subject "CN=QX Project Dev" -CertStoreLocation Cert:\CurrentUser\My

# Export PFX if needed for CI (dev only — never commit)
Export-PfxCertificate -Cert $cert -FilePath qx-dev-codesign.pfx -Password (Read-Host -AsSecureString)

# Sign
signtool sign /fd SHA256 /a /f qx-dev-codesign.pfx /p <password> bin\qx-launcher.exe
```

Still submit false positives to Microsoft for **unsigned** or **self-signed** builds if Defender blocks them.

### 3.4 CI integration (optional)

Store the `.pfx` in GitHub Actions secrets and add a signing step to `.github/workflows/prod-release.yml` **before** the binary is copied into the web image. Never commit the private key.

---

## 4. Report a false positive to Microsoft

If Defender still blocks a **signed** or **latest** build:

1. **Windows Security → Protection history** — note detection name, file path, and hash.
2. Submit the file: [Microsoft Security Intelligence — Submit a file](https://www.microsoft.com/en-us/wdsi/filesubmission) — choose **Software developer** / false positive.
3. Use the template below (copy/paste and fill in).

Microsoft typically re-evaluates within 1–3 business days.

For enterprise: [Submit via MDE portal](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/defender-endpoint-false-positives-negatives) if applicable.

### 4.1 Submission template

```text
Submission type: Software developer — false positive

Product name: QX Launcher
Publisher: QX Project
File name: qx-launcher.exe
Detection name: Program:Win32/Wacapew.A!ml  (or exact name from Protection history)

Official download URL:
  https://mc.qx-dev.ru/downloads/qx-launcher.exe

SHA-256 of the submitted file:
  <paste hash from §5>

Version / git tag:
  <e.g. v0.2.0 or git describe output>

Description:
  QX Launcher is a legitimate Minecraft launcher tray daemon for the QX Project
  platform. It links a user device to mc.qx-dev.ru, syncs game instances, and
  launches Minecraft with the correct Java runtime and mods.

  It is NOT malware. Mitigations already in the binary:
  - Self-update uses in-place rename (qx-launcher.exe → .prev) with no batch/cmd helper
  - Updates are staged under %LOCALAPPDATA%\QXLauncher\updates
  - Desktop notifications use WinRT COM (no PowerShell)
  - URLs open via ShellExecute, not rundll32
  - PE version info and asInvoker manifest are embedded at build

  Persistence: optional user startup entry only (user choice).

Contact: <your email>
```

### 4.2 User workaround (until reputation improves)

- **Windows Security → Virus & threat protection → Manage settings → Exclusions** — add the folder containing `qx-launcher.exe` (user choice; document in FAQ).
- Or: Protection history → **Allow** on the specific detection (per-event).

---

## 5. Verify build locally

### 5.1 PE version metadata

After building:

```powershell
(Get-Item bin\qx-launcher.exe).VersionInfo | Format-List *
```

Expect `CompanyName`, `ProductName`, `FileDescription`, and `OriginalFilename` populated.

### 5.2 SHA-256 hash (for Microsoft submission)

```powershell
Get-FileHash bin\qx-launcher.exe -Algorithm SHA256 | Format-List
```

On Linux/macOS CI artifacts:

```bash
sha256sum bin/qx-launcher.exe
```

---

## 6. Future: MSI / WiX distribution (not implemented)

An **MSI installer** signed with the same publisher certificate is the standard pattern for Windows desktop apps and can avoid “replace running EXE” heuristics entirely:

1. Build `qx-launcher.exe` (signed).
2. Package with WiX (`MajorUpgrade`, per-user or per-machine install dir).
3. Tray app checks for updates via MSI version / API; installer runs `msiexec /i` (elevated only if per-machine).

QX Launcher today ships as a **single portable EXE** with in-place self-update for MVP. MSI is documented here as the recommended production path when signing budget allows; no WiX project exists in this repo yet.

---

*See also [auto-update.md](./auto-update.md) (signing roadmap), [security-legal.md](./security-legal.md) §10.*

Last updated: 2026-07-01
