import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { PageMeta } from '@/components/PageMeta';
import { useI18n } from '@/i18n/I18nContext';
import {
  absoluteUrl,
  buildCanonicalPath,
  getSiteUrl,
  resolveSeoPageKey,
  shouldNoIndex,
} from '@/lib/seo';

export function RouteSeo() {
  const { pathname } = useLocation();
  const { locale, t } = useI18n();
  const pageKey = resolveSeoPageKey(pathname);
  const canonicalPath = buildCanonicalPath(pathname);
  const siteUrl = getSiteUrl();
  const siteName = t('seo.siteName');

  const jsonLd = useMemo(() => {
    if (pathname !== '/' && !pathname.startsWith('/launcher')) {
      return undefined;
    }

    const graph: Array<Record<string, unknown>> = [];

    if (pathname === '/') {
      graph.push(
        {
          '@type': 'WebSite',
          '@id': `${siteUrl}/#website`,
          name: siteName,
          url: siteUrl || undefined,
          description: t('seo.defaultDescription'),
          inLanguage: ['ru', 'en'],
        },
        {
          '@type': 'Organization',
          '@id': `${siteUrl}/#organization`,
          name: siteName,
          url: siteUrl || undefined,
          logo: siteUrl ? absoluteUrl('/favicon.svg', siteUrl) : '/favicon.svg',
        },
      );
    }

    if (pathname === '/' || pathname.startsWith('/launcher')) {
      graph.push({
        '@type': 'SoftwareApplication',
        name: 'QXLauncher',
        applicationCategory: 'GameApplication',
        operatingSystem: 'Windows, macOS, Linux',
        description: t('seo.jsonLd.launcherDescription'),
        url: siteUrl ? absoluteUrl('/launcher', siteUrl) : '/launcher',
        inLanguage: locale,
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'USD',
        },
      });
    }

    if (graph.length === 0) {
      return undefined;
    }

    return {
      '@context': 'https://schema.org',
      '@graph': graph,
    };
  }, [locale, pathname, siteName, siteUrl, t]);

  return (
    <PageMeta
      titleKey={`seo.pages.${pageKey}.title`}
      descriptionKey={`seo.pages.${pageKey}.description`}
      pathname={canonicalPath}
      noIndex={shouldNoIndex(pathname)}
      jsonLd={jsonLd}
    />
  );
}
