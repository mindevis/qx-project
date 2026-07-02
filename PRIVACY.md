# QXSystem Privacy Policy

**Effective date:** 2026-07-02  
**Contact:** mindevis.by@gmail.com

Full documentation: [docs.qx-dev.ru/privacy](https://docs.qx-dev.ru/privacy/)

QXSystem (“we”, “the project”) operates [mc.qx-dev.ru](https://mc.qx-dev.ru) and related open-source software (QXWeb, QXApi, QXLauncher, QXAgent). This policy describes what data we process and why.

## What we collect

| Data | Where | Purpose |
| ---- | ----- | ------- |
| Account email and password hash | QXApi | Authentication |
| JWT / refresh tokens | Browser / QXLauncher local storage | Session |
| Device ID and launcher version | QXLauncher ↔ QXApi | Link desktop launcher to your account |
| Minecraft / Microsoft OAuth tokens (if you link) | QXApi (encrypted at rest where configured) | Licensed launch and server connect |
| Game instances, mods metadata, server config | QXApi database | Core product features |
| IP address, request logs | Web server / API | Security, rate limiting, abuse prevention |

We do **not** sell personal data.

## QXLauncher (Windows)

QXLauncher may store tokens and device identifiers locally on your PC (`%LOCALAPPDATA%\QXLauncher`). It downloads Minecraft-related files and launcher updates from our API/CDN. Optional self-update replaces only the QX Launcher binary after you request an update.

## Retention

Account data is kept while your account exists. Logs are rotated per server configuration (typically days to weeks).

## Your rights

You may request access or deletion of your account data by emailing **mindevis.by@gmail.com**.

## Changes

We may update this policy; the latest version is published at [docs.qx-dev.ru/privacy](https://docs.qx-dev.ru/privacy/).

## Open source

Source code: [github.com/mindevis/qx-project](https://github.com/mindevis/qx-project) (MIT License).
