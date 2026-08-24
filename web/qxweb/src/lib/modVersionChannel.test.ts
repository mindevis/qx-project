import { describe, expect, it } from 'vitest';
import { catalogProjectUrl, inferModVersionChannel, resolveModVersionChannel } from './modVersionChannel';

describe('resolveModVersionChannel', () => {
  it('prefers an explicit catalog type', () => {
    expect(resolveModVersionChannel('alpha', '1.0.0')).toBe('alpha');
    expect(resolveModVersionChannel('release', '1.0.0-beta')).toBe('release');
  });

  it('infers alpha and beta from names', () => {
    expect(inferModVersionChannel('mod-1.2.0-alpha.1.jar')).toBe('alpha');
    expect(inferModVersionChannel('plugin-2.0.0-rc.1.jar')).toBe('beta');
    expect(inferModVersionChannel('sodium-0.6.0.jar')).toBe('release');
  });
});

describe('catalogProjectUrl', () => {
  it('builds vendor pages', () => {
    expect(catalogProjectUrl('modrinth', 'sodium', 'sodium')).toBe('https://modrinth.com/project/sodium');
    expect(catalogProjectUrl('curseforge', '238222')).toBe('https://www.curseforge.com/projects/238222');
  });
});
