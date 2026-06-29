# SEO (QXWeb)

QXWeb is a React SPA (Vite). Search engines receive static fallbacks from `index.html`, build-time `robots.txt` / `sitemap.xml`, and per-route meta tags updated at runtime.

## Production configuration

Set the public site origin before building or deploying:

```bash
# web.toml (repo root, preferred — see web.toml.example)
site_url = "https://mc.qx-dev.ru"

# or environment / CI
export VITE_SITE_URL=https://mc.qx-dev.ru
```

`VITE_SITE_URL` drives:

- `<link rel="canonical">` and Open Graph `og:url` / `og:image`
- `dist/robots.txt` → `Sitemap: …/sitemap.xml`
- `dist/sitemap.xml` absolute URLs

If unset, the build defaults to `https://mc.qx-dev.ru`. In the browser, canonical URLs fall back to `window.location.origin`.

## Per-route meta

`RouteSeo` (in `AppLayout`) maps routes to i18n keys under `seo.pages.*` in `web/qxweb/src/i18n/locales/{ru,en}.ts`.

Titles use the pattern `{page title} | QXSystem`. Auth routes (`/auth/*`) are marked `noindex`.

JSON-LD:

- Home (`/`): `WebSite` + `Organization`
- Launcher (`/launcher`): `SoftwareApplication` for QXLauncher

## Updating the sitemap

Main routes are listed in `web/qxweb/vite.seo.ts` (`SITEMAP_ROUTES`). After adding a public landing page:

1. Add the path to `SITEMAP_ROUTES` in `vite.seo.ts` (and optionally `src/lib/seo.ts` `SITEMAP_ROUTES` for reference).
2. Bump `SITEMAP_LASTMOD` in `vite.seo.ts` to the release date.
3. Add `seo.pages.<key>.title` / `description` in both locale files.
4. Rebuild: `npm run build` in `web/qxweb`.

Dev server serves `/robots.txt` and `/sitemap.xml` dynamically with the same logic.

## Static assets

| File | Purpose |
|------|---------|
| `public/favicon.svg`, `favicon.ico` | Favicon |
| `public/og-image.svg` | Default Open Graph / Twitter image (1200×630 SVG) |

For best social previews, consider exporting `og-image.svg` to PNG and updating `OG_IMAGE_PATH` in `src/lib/seo.ts`.

## Crawlers and JavaScript

There is no SSR/prerender yet. `index.html` includes Russian default meta for the home page; `HomePage` adds a `<noscript>` summary. Google and Yandex execute JavaScript for SPAs, but prerender or SSR remains a future improvement for faster indexing.

## Search console checklist

1. Deploy with correct `VITE_SITE_URL`.
2. Verify `https://<domain>/robots.txt` and `https://<domain>/sitemap.xml`.
3. Submit the sitemap in [Google Search Console](https://search.google.com/search-console) and [Yandex Webmaster](https://webmaster.yandex.ru/).
4. Request indexing for `/`, `/launcher`, and `/monitoring`.
5. Monitor Core Web Vitals and mobile usability in Search Console.
