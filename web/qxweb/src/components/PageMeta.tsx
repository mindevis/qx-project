import { useEffect, useMemo } from 'react';
import { useI18n } from '@/i18n/I18nContext';
import {
  absoluteUrl,
  formatPageTitle,
  getSiteUrl,
  localeOpenGraphCode,
  OG_IMAGE_PATH,
  removeLink,
  removeMeta,
  upsertLink,
  upsertMeta,
} from '@/lib/seo';

type PageMetaProps = {
  titleKey: string;
  descriptionKey: string;
  pathname: string;
  noIndex?: boolean;
  jsonLd?: Record<string, unknown> | Array<Record<string, unknown>>;
};

export function PageMeta({
  titleKey,
  descriptionKey,
  pathname,
  noIndex = false,
  jsonLd,
}: PageMetaProps) {
  const { locale, t } = useI18n();
  const siteName = t('seo.siteName');
  const pageTitle = t(titleKey);
  const description = t(descriptionKey);
  const documentTitle = formatPageTitle(pageTitle, siteName);
  const canonicalPath = pathname || '/';
  const siteUrl = getSiteUrl();
  const canonicalUrl = absoluteUrl(canonicalPath, siteUrl);
  const imageUrl = absoluteUrl(OG_IMAGE_PATH, siteUrl);
  const ogLocale = localeOpenGraphCode(locale);
  const alternateLocale = locale === 'en' ? 'ru_RU' : 'en_US';

  const jsonLdPayload = useMemo(() => {
    if (!jsonLd) {
      return null;
    }
    return JSON.stringify(jsonLd);
  }, [jsonLd]);

  useEffect(() => {
    document.title = documentTitle;
    upsertMeta('name', 'description', description);
    upsertMeta('property', 'og:type', 'website');
    upsertMeta('property', 'og:site_name', siteName);
    upsertMeta('property', 'og:title', documentTitle);
    upsertMeta('property', 'og:description', description);
    upsertMeta('property', 'og:image', imageUrl);
    upsertMeta('property', 'og:locale', ogLocale);
    upsertMeta('property', 'og:locale:alternate', alternateLocale);
    upsertMeta('name', 'twitter:card', 'summary_large_image');
    upsertMeta('name', 'twitter:title', documentTitle);
    upsertMeta('name', 'twitter:description', description);
    upsertMeta('name', 'twitter:image', imageUrl);

    if (noIndex) {
      upsertMeta('name', 'robots', 'noindex, nofollow');
      removeLink('canonical');
      removeMeta('property', 'og:url');
    } else {
      removeMeta('name', 'robots');
      upsertLink('canonical', canonicalUrl);
      upsertMeta('property', 'og:url', canonicalUrl);
    }

    return () => {
      removeMeta('name', 'robots');
    };
  }, [
    alternateLocale,
    canonicalUrl,
    description,
    documentTitle,
    imageUrl,
    noIndex,
    ogLocale,
    siteName,
  ]);

  if (!jsonLdPayload) {
    return null;
  }

  return (
    <script
      type="application/ld+json"
      data-qx-seo="jsonld"
      dangerouslySetInnerHTML={{ __html: jsonLdPayload }}
    />
  );
}
