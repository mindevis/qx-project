export const OG_IMAGE_PATH = '/og-image.svg';

export const SITEMAP_ROUTES = [
  { path: '/', changefreq: 'weekly', priority: '1.0' },
  { path: '/launcher', changefreq: 'weekly', priority: '0.9' },
  { path: '/launcher/link', changefreq: 'monthly', priority: '0.6' },
  { path: '/monitoring', changefreq: 'daily', priority: '0.8' },
  { path: '/servers', changefreq: 'weekly', priority: '0.7' },
  { path: '/profile', changefreq: 'monthly', priority: '0.5' },
] as const;

const SEO_PAGE_KEYS = [
  'home',
  'launcher',
  'launcherLink',
  'monitoring',
  'servers',
  'profile',
  'auth',
] as const;

export type SeoPageKey = (typeof SEO_PAGE_KEYS)[number];

export function getSiteUrl(): string {
  const configured = import.meta.env.VITE_SITE_URL?.trim().replace(/\/$/, '');
  if (configured) {
    return configured;
  }
  if (typeof window !== 'undefined') {
    return window.location.origin;
  }
  return '';
}

export function formatPageTitle(pageTitle: string, siteName: string): string {
  const trimmed = pageTitle.trim();
  if (!trimmed || trimmed === siteName || trimmed.startsWith(`${siteName} —`)) {
    return trimmed || siteName;
  }
  return `${trimmed} | ${siteName}`;
}

export function resolveSeoPageKey(pathname: string): SeoPageKey {
  if (pathname === '/') return 'home';
  if (pathname === '/launcher/link') return 'launcherLink';
  if (pathname.startsWith('/launcher')) return 'launcher';
  if (pathname === '/monitoring') return 'monitoring';
  if (pathname.startsWith('/servers')) return 'servers';
  if (pathname === '/profile') return 'profile';
  if (pathname.startsWith('/auth')) return 'auth';
  return 'home';
}

export function shouldNoIndex(pathname: string): boolean {
  return pathname.startsWith('/auth');
}

export function buildCanonicalPath(pathname: string): string {
  if (pathname.startsWith('/auth')) {
    return '/';
  }
  if (pathname.startsWith('/servers/')) {
    return '/servers';
  }
  if (pathname.startsWith('/launcher/') && pathname !== '/launcher/link') {
    return '/launcher';
  }
  return pathname || '/';
}

export function absoluteUrl(path: string, siteUrl = getSiteUrl()): string {
  if (!siteUrl) {
    return path;
  }
  return `${siteUrl}${path.startsWith('/') ? path : `/${path}`}`;
}

export function localeOpenGraphCode(locale: string): string {
  return locale === 'en' ? 'en_US' : 'ru_RU';
}

export function upsertMeta(
  attribute: 'name' | 'property',
  key: string,
  content: string,
): HTMLMetaElement {
  const selector = `meta[${attribute}="${key}"]`;
  let element = document.head.querySelector<HTMLMetaElement>(selector);
  if (!element) {
    element = document.createElement('meta');
    element.setAttribute(attribute, key);
    document.head.appendChild(element);
  }
  element.setAttribute('content', content);
  return element;
}

export function upsertLink(rel: string, href: string): HTMLLinkElement {
  const selector = `link[rel="${rel}"]`;
  let element = document.head.querySelector<HTMLLinkElement>(selector);
  if (!element) {
    element = document.createElement('link');
    element.setAttribute('rel', rel);
    document.head.appendChild(element);
  }
  element.setAttribute('href', href);
  return element;
}

export function removeMeta(attribute: 'name' | 'property', key: string): void {
  document.head.querySelector(`meta[${attribute}="${key}"]`)?.remove();
}

export function removeLink(rel: string): void {
  document.head.querySelector(`link[rel="${rel}"]`)?.remove();
}
