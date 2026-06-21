import type { Messages } from './locales/ru';
import { en } from './locales/en';
import { ru } from './locales/ru';

export type Locale = 'ru' | 'en';

export const LOCALE_STORAGE_KEY = 'qxweb-locale';
export const DEFAULT_LOCALE: Locale = 'ru';

const catalogs: Record<Locale, Messages> = { ru, en };

function getNested(obj: Record<string, unknown>, path: string): string | undefined {
  let current: unknown = obj;
  for (const part of path.split('.')) {
    if (typeof current !== 'object' || current === null || !(part in current)) {
      return undefined;
    }
    current = (current as Record<string, unknown>)[part];
  }
  return typeof current === 'string' ? current : undefined;
}

export function readStoredLocale(): Locale {
  /* v8 ignore next 3 -- @preserve */
  if (typeof window === 'undefined') {
    return DEFAULT_LOCALE;
  }
  const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY);
  return stored === 'en' || stored === 'ru' ? stored : DEFAULT_LOCALE;
}

export function translate(
  locale: Locale,
  key: string,
  params?: Record<string, string | number>,
): string {
  const template = getNested(catalogs[locale] as unknown as Record<string, unknown>, key)
    ?? getNested(catalogs[DEFAULT_LOCALE] as unknown as Record<string, unknown>, key)
    ?? key;

  if (!params) {
    return template;
  }

  return Object.entries(params).reduce(
    (text, [name, value]) => text.replaceAll(`{{${name}}}`, String(value)),
    template,
  );
}

export function getLaunchStatusKey(status: string): string {
  return `launcher.launchStatus.${status}`;
}

export function getServerStatusKey(status: string): string {
  return `servers.status.${status}`;
}
