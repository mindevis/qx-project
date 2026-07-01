# Web

| Папка | Компонент |
| ------- | ----------- |
| [qxweb](./qxweb/) | **QXWeb** — React SPA (панель + `/launcher`) |

Frontend отделён от Go-сервисов в `services/`.

## QXWeb

**Стек:** TypeScript, React, Vite, Ant Design 6.4.5. **Node.js 24+**

**Phase 0–3 + Alpha:** `/`, auth, `/profile`, `/launcher`, **`/servers`**, **`/servers/:id/game-servers/:id`**.

**Prod:** панель **[mc.qx-dev.ru](https://mc.qx-dev.ru)** — live с 2026-06-29 — [production-deploy.md](../docs/production-deploy.md).

```bash
cp ../../web.toml.example ../../web.toml   # из корня репо
cd qxweb
npm install
npm run dev          # http://localhost:5173
npm run test         # Vitest
npm run test:coverage  # 100% thresholds
npm run build
```

**Конфиг (dev):** корневой `web.toml` — см. [docs/configuration.md](../docs/configuration.md).

| Key | Описание |
| ----- | ---------- |
| `api_base_url` | REST base (мапится в `VITE_API_BASE_URL`) |
| `log_level` | Уровень логов фронтенда |
| `launcher_download_url` | URL кнопки «Скачать QXLauncher» |

Vite proxy: `/api` → `http://localhost:3000` (`vite.config.ts`).

## SEO (production build)

Set public site origin before build:

```bash
# web.toml (repo root)
site_url = "https://mc.qx-dev.ru"

# or CI / shell
export VITE_SITE_URL=https://mc.qx-dev.ru
```

This drives `robots.txt`, `sitemap.xml`, canonical URLs, and Open Graph tags. Per-route meta is in `RouteSeo` + i18n keys `seo.pages.*`.
