import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { api } from '@/api/client';
import { fetchMissingResourceIcons, instanceResourceIconKey } from './instanceResourceIcons';
import { clearModCatalogCaches } from './modCatalogCache';

describe('instanceResourceIconKey', () => {
  it('builds stable project key', () => {
    expect(instanceResourceIconKey({ source: 'curseforge', project_id: '123' })).toBe('curseforge:123');
  });
});

describe('fetchMissingResourceIcons', () => {
  beforeEach(() => {
    clearModCatalogCaches();
    vi.spyOn(api, 'getModProject').mockResolvedValue({
      id: 'dep-1',
      source: 'curseforge',
      slug: 'fabric-api',
      name: 'Fabric API',
      icon_url: 'https://example/icon.png',
      external_url: 'https://example.com',
      project_type: 'mod',
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    clearModCatalogCaches();
  });

  it('fetches icons only for resources without icon_url', async () => {
    const icons = await fetchMissingResourceIcons([
      {
        source: 'curseforge',
        project_id: 'dep-1',
        project_name: 'Fabric API',
        filename: 'fabric-api.jar',
        resource_type: 'mod',
        installed_at: 'now',
        icon_url: '',
      },
      {
        source: 'curseforge',
        project_id: 'main-1',
        project_name: 'Main',
        filename: 'main.jar',
        resource_type: 'mod',
        installed_at: 'now',
        icon_url: 'https://example/main.png',
      },
    ]);

    expect(api.getModProject).toHaveBeenCalledTimes(1);
    expect(icons).toEqual({ 'curseforge:dep-1': 'https://example/icon.png' });
  });
});
