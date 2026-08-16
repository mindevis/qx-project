import { describe, expect, it, vi, beforeEach } from 'vitest';
import { api } from '@/api/client';
import { attachCatalogPartners, clearCatalogPartnerCache } from './catalogPartners';

describe('attachCatalogPartners', () => {
  beforeEach(() => {
    clearCatalogPartnerCache();
    vi.restoreAllMocks();
  });

  it('adds the other source when search finds the same slug', async () => {
    vi.spyOn(api, 'searchMods').mockResolvedValue({
      items: [
        {
          source: 'curseforge',
          id: '263420',
          slug: 'xaeros-minimap',
          name: "Xaero's Minimap",
          project_type: 'mod',
          external_url: 'https://www.curseforge.com/minecraft/mc-mods/xaeros-minimap',
        },
      ],
      curseforge_enabled: true,
    });

    const items = await attachCatalogPartners(
      [
        {
          source: 'modrinth',
          id: 'xaero',
          slug: 'xaeros-minimap',
          name: "Xaero's Minimap",
          project_type: 'mod',
          external_url: 'https://modrinth.com/mod/xaeros-minimap',
        },
      ],
      { loader: 'neoforge', mcVersion: '1.21.1', type: 'mod' },
    );

    expect(items).toHaveLength(2);
    expect(items.map((item) => item.source)).toEqual(['modrinth', 'curseforge']);
    expect(api.searchMods).toHaveBeenCalledWith(
      expect.objectContaining({ q: 'xaeros-minimap', source: 'curseforge' }),
    );
  });
});
