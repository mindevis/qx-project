import { describe, expect, it } from 'vitest';
import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  getLaunchStatusKey,
  getServerStatusKey,
  readStoredLocale,
  translate,
} from './index';

describe('i18n translate', () => {
  it('returns Russian strings by default', () => {
    expect(translate('ru', 'home.title')).toBe('Единая экосистема для Minecraft');
  });

  it('returns English strings', () => {
    expect(translate('en', 'home.title')).toBe('Unified ecosystem for Minecraft');
  });

  it('interpolates params', () => {
    expect(translate('ru', 'placeholder.body', { phase: 'P3' })).toContain('P3');
  });

  it('falls back to default locale then key', () => {
    expect(translate('en', 'missing.key')).toBe('missing.key');
  });

  it('returns key when nested value is not a string', () => {
    expect(translate('ru', 'launcher.launchStatus')).toBe('launcher.launchStatus');
    expect(translate('ru', 'launcher')).toBe('launcher');
  });

  it('reads stored locale', () => {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, 'en');
    expect(readStoredLocale()).toBe('en');
    window.localStorage.removeItem(LOCALE_STORAGE_KEY);
    expect(readStoredLocale()).toBe(DEFAULT_LOCALE);
    window.localStorage.setItem(LOCALE_STORAGE_KEY, 'bad');
    expect(readStoredLocale()).toBe(DEFAULT_LOCALE);
  });

  it('builds status keys', () => {
    expect(getLaunchStatusKey('queued')).toBe('launcher.launchStatus.queued');
    expect(getLaunchStatusKey('preparing')).toBe('launcher.launchStatus.preparing');
    expect(translate('ru', 'launcher.launchStatus.downloading')).toContain('Скачивание');
    expect(getServerStatusKey('online')).toBe('servers.status.online');
  });
});
