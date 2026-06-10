# Web

| Папка | Компонент |
| ------- | ----------- |
| [qxweb](./qxweb/) | **QXWeb** — React SPA (панель + `/launcher`) |

Frontend отделён от Go-сервисов в `services/`.

## QXWeb

**Стек:** TypeScript, React, Vite, Ant Design.

**Phase 0:** `/` (лендинг), модалка входа/регистрации (`/auth/:mode` → redirect), `/profile` (email + смена email/пароля в модалках), `/launcher` (UI Phase 1), placeholder `/servers`. Тема light/dark, меню аккаунта в шапке.

```bash
cd qxweb
npm install
npm run dev          # http://localhost:5173
npm run test         # Vitest
npm run test:coverage  # 100% thresholds
npm run build
```

API base (dev): `http://localhost:3000/api/v1` (`VITE_API_BASE_URL` в `.env`).  
Vite proxy: `/api` → `http://localhost:3000` (`vite.config.ts`).
