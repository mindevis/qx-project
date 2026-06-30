import type { Plugin } from 'vite';

const DEFAULT_SITE_URL = 'https://mc.qx-dev.ru';
const SITEMAP_LASTMOD = '2026-06-29';

const SITEMAP_ROUTES = [
  { path: '/', changefreq: 'weekly', priority: '1.0' },
  { path: '/launcher', changefreq: 'weekly', priority: '0.9' },
  { path: '/launcher/link', changefreq: 'monthly', priority: '0.6' },
  { path: '/monitoring', changefreq: 'daily', priority: '0.8' },
  { path: '/servers', changefreq: 'weekly', priority: '0.7' },
  { path: '/profile', changefreq: 'monthly', priority: '0.5' },
  { path: '/skins', changefreq: 'monthly', priority: '0.5' },
] as const;

function normalizeSiteUrl(raw?: string): string {
  return (raw?.trim() || DEFAULT_SITE_URL).replace(/\/$/, '');
}

function buildRobotsTxt(siteUrl: string): string {
  return [`User-agent: *`, `Allow: /`, ``, `Sitemap: ${siteUrl}/sitemap.xml`, ``].join('\n');
}

function buildSitemapXml(siteUrl: string): string {
  const urls = SITEMAP_ROUTES.map(
    (route) => `  <url>
    <loc>${siteUrl}${route.path}</loc>
    <lastmod>${SITEMAP_LASTMOD}</lastmod>
    <changefreq>${route.changefreq}</changefreq>
    <priority>${route.priority}</priority>
  </url>`,
  ).join('\n');

  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls}
</urlset>
`;
}

function apiOrigin(apiBaseUrl?: string): string | null {
  if (!apiBaseUrl || apiBaseUrl.startsWith('/')) {
    return null;
  }
  try {
    return new URL(apiBaseUrl).origin;
  } catch {
    return null;
  }
}

export function seoStaticPlugin(): Plugin {
  const siteUrl = normalizeSiteUrl(process.env.VITE_SITE_URL);
  const apiPreconnect = apiOrigin(process.env.VITE_API_BASE_URL);

  return {
    name: 'qx-seo-static',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (req.url === '/robots.txt') {
          res.setHeader('Content-Type', 'text/plain; charset=utf-8');
          res.end(buildRobotsTxt(siteUrl));
          return;
        }
        if (req.url === '/sitemap.xml') {
          res.setHeader('Content-Type', 'application/xml; charset=utf-8');
          res.end(buildSitemapXml(siteUrl));
          return;
        }
        next();
      });
    },
    generateBundle() {
      this.emitFile({
        type: 'asset',
        fileName: 'robots.txt',
        source: buildRobotsTxt(siteUrl),
      });
      this.emitFile({
        type: 'asset',
        fileName: 'sitemap.xml',
        source: buildSitemapXml(siteUrl),
      });
    },
    transformIndexHtml(html) {
      const preconnect = apiPreconnect
        ? `\n    <link rel="preconnect" href="${apiPreconnect}" crossorigin />`
        : '';
      const canonical = `\n    <link rel="canonical" href="${siteUrl}/" />`;
      const ogImage = `${siteUrl}/og-image.svg`;

      return html
        .replace('</head>', `${preconnect}${canonical}\n  </head>`)
        .replace(
          '<meta name="viewport"',
          `<meta name="description" content="QXSystem — единая экосистема для Minecraft: QXLauncher, QXMods и QXAgent на одном аккаунте." />
    <meta name="robots" content="index, follow" />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="QXSystem" />
    <meta property="og:title" content="QXSystem — экосистема Minecraft" />
    <meta property="og:description" content="Десктопный лаунчер, каталог модов и агент для вашего сервера — один аккаунт и общие настройки." />
    <meta property="og:url" content="${siteUrl}/" />
    <meta property="og:image" content="${ogImage}" />
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:title" content="QXSystem — экосистема Minecraft" />
    <meta name="twitter:description" content="Десктопный лаунчер, каталог модов и агент для вашего сервера — один аккаунт и общие настройки." />
    <meta name="twitter:image" content="${ogImage}" />
    <meta name="theme-color" content="#1677ff" />
    <meta name="viewport"`,
        );
    },
  };
}
