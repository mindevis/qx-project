import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { api } from '@/api/client';
import {
  cachedGetModProject,
  cachedListModVersions,
  clearModCatalogCaches,
} from './modCatalogCache';

describe('modCatalogCache', () => {
  beforeEach(() => {
    clearModCatalogCaches();
    vi.spyOn(api, 'getModProject').mockResolvedValue({
      id: 'sodium',
      source: 'modrinth',
      slug: 'sodium',
      name: 'Sodium',
      external_url: 'https://modrinth.com/mod/sodium',
      project_type: 'mod',
      icon_url: 'https://cdn/icon.png',
    });
    vi.spyOn(api, 'listModVersions').mockResolvedValue({
      items: [{ id: 'v1', version_number: '1.0', files: [{ filename: 'a.jar', url: 'https://x' }] }],
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    clearModCatalogCaches();
  });

  it('deduplicates mod project lookups', async () => {
    await cachedGetModProject('modrinth', 'sodium');
    await cachedGetModProject('modrinth', 'sodium');
    expect(api.getModProject).toHaveBeenCalledTimes(1);
  });

  it('deduplicates mod version list lookups', async () => {
    await cachedListModVersions('modrinth', 'sodium', { loader: 'fabric', mc_version: '1.21' });
    await cachedListModVersions('modrinth', 'sodium', { loader: 'fabric', mc_version: '1.21' });
    expect(api.listModVersions).toHaveBeenCalledTimes(1);
  });
});
