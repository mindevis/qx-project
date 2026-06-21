# Web

| Папка | Компонент |
| ------- | ----------- |
| [qxweb](./qxweb/) | **QXWeb** — React SPA (панель + `/launcher`) |

Frontend отделён от Go-сервисов в `services/`.

## QXWeb

**Стек:** TypeScript, React, Vite, Ant Design.

**Phase 0–3 + Alpha (dev):** `/`, auth, `/profile`, `/launcher`, **`/servers`**. Prod 🔲.

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
