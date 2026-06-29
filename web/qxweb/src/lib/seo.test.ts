import { describe, expect, it } from 'vitest';
import {
  absoluteUrl,
  buildCanonicalPath,
  formatPageTitle,
  localeOpenGraphCode,
  resolveSeoPageKey,
  shouldNoIndex,
} from './seo';

describe('seo helpers', () => {
  it('formats page titles with site name', () => {
    expect(formatPageTitle('Лаунчер', 'QXSystem')).toBe('Лаунчер | QXSystem');
    expect(formatPageTitle('QXSystem — экосистема Minecraft', 'QXSystem')).toBe(
      'QXSystem — экосистема Minecraft',
    );
  });

  it('resolves SEO page keys from pathname', () => {
    expect(resolveSeoPageKey('/')).toBe('home');
    expect(resolveSeoPageKey('/launcher')).toBe('launcher');
    expect(resolveSeoPageKey('/launcher/link')).toBe('launcherLink');
    expect(resolveSeoPageKey('/monitoring')).toBe('monitoring');
    expect(resolveSeoPageKey('/servers/abc')).toBe('servers');
    expect(resolveSeoPageKey('/profile')).toBe('profile');
    expect(resolveSeoPageKey('/auth/login')).toBe('auth');
  });

  it('marks auth routes as noindex', () => {
    expect(shouldNoIndex('/auth/login')).toBe(true);
    expect(shouldNoIndex('/launcher')).toBe(false);
  });

  it('builds canonical paths for nested routes', () => {
    expect(buildCanonicalPath('/servers/abc/game-servers/1')).toBe('/servers');
    expect(buildCanonicalPath('/launcher/instances')).toBe('/launcher');
    expect(buildCanonicalPath('/launcher/link')).toBe('/launcher/link');
    expect(buildCanonicalPath('/auth/login')).toBe('/');
  });

  it('builds absolute URLs', () => {
    expect(absoluteUrl('/launcher', 'https://mc.qx-dev.ru')).toBe('https://mc.qx-dev.ru/launcher');
  });

  it('maps locale to Open Graph locale codes', () => {
    expect(localeOpenGraphCode('ru')).toBe('ru_RU');
    expect(localeOpenGraphCode('en')).toBe('en_US');
  });
});
