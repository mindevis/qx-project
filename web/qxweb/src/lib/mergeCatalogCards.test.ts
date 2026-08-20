import { describe, expect, it } from 'vitest';
import type { ModCatalogItem } from '@/api/client';
import {
  catalogCardItem,
  catalogItemNameKey,
  mergeCatalogCardsByName,
  preferredCatalogItem,
} from './mergeCatalogCards';

function item(partial: Partial<ModCatalogItem> & Pick<ModCatalogItem, 'id' | 'source' | 'name'>): ModCatalogItem {
  return {
    slug: partial.slug ?? partial.id,
    summary: '',
    project_type: 'mod',
    external_url: partial.external_url ?? '',
    ...partial,
  };
}

describe('mergeCatalogCardsByName', () => {
  const jeiMr = item({
    id: 'jei',
    source: 'modrinth',
    name: 'JEI',
    downloads: 100,
    external_url: 'https://modrinth.com/mod/jei',
  });
  const jeiCf = item({
    id: '238222',
    source: 'curseforge',
    name: 'jei',
    downloads: 80,
    external_url: 'https://www.curseforge.com/minecraft/mc-mods/jei',
  });
  const sodium = item({
    id: 'sodium',
    source: 'modrinth',
    name: 'Sodium',
    downloads: 200,
  });

  it('normalizes names for matching', () => {
    expect(catalogItemNameKey('  Just   Enough Items ')).toBe('just enough items');
  });

  it('merges same-name mods from both sources when the filter is all', () => {
    const cards = mergeCatalogCardsByName([jeiMr, sodium, jeiCf], 'all');
    expect(cards).toHaveLength(2);
    expect(cards[0].name).toBe('JEI');
    expect(cards[0].items.map((row) => row.source)).toEqual(['modrinth', 'curseforge']);
    expect(cards[1].items).toEqual([sodium]);
  });

  it('does not merge when a single source is selected', () => {
    const cards = mergeCatalogCardsByName([jeiMr, jeiCf], 'modrinth');
    expect(cards).toHaveLength(2);
    expect(cards.map((card) => card.items)).toEqual([[jeiMr], [jeiCf]]);
  });

  it('prefers the listing with more downloads', () => {
    expect(preferredCatalogItem([jeiCf, jeiMr])).toBe(jeiMr);
    expect(catalogCardItem({ key: 'k', name: 'JEI', items: [jeiMr, jeiCf] }, 'curseforge')).toBe(jeiCf);
  });

  it('leaves leftover same-name listings as their own cards', () => {
    const extraMr = item({ id: 'jei-fork', source: 'modrinth', name: 'JEI', downloads: 1 });
    const cards = mergeCatalogCardsByName([jeiMr, extraMr, jeiCf], 'all');
    expect(cards).toHaveLength(2);
    expect(cards[0].items).toHaveLength(2);
    expect(cards[1].items).toEqual([extraMr]);
  });

  it('merges when a parenthetical suffix is the only name difference', () => {
    const cards = mergeCatalogCardsByName(
      [
        item({ id: 'yacl', source: 'modrinth', slug: 'yacl', name: 'YetAnotherConfigLib (YACL)' }),
        item({
          id: '667299',
          source: 'curseforge',
          slug: 'yetanotherconfiglib',
          name: 'YetAnotherConfigLib',
        }),
      ],
      'all',
    );
    expect(cards).toHaveLength(1);
    expect(cards[0].items).toHaveLength(2);
  });

  it('merges plugin listings from more than two catalogs onto one card', () => {
    const cards = mergeCatalogCardsByName(
      [
        item({ id: 'protocollib', source: 'modrinth', slug: 'protocollib', name: 'ProtocolLib' }),
        item({ id: '1997', source: 'spigot', slug: 'protocollib', name: 'ProtocolLib' }),
        item({ id: 'ProtocolLib', source: 'hangar', slug: 'ProtocolLib', name: 'ProtocolLib' }),
        item({ id: '15590', source: 'bukkit', slug: 'protocollib', name: 'ProtocolLib' }),
      ],
      'all',
    );
    expect(cards).toHaveLength(1);
    expect(cards[0].items.map((row) => row.source)).toEqual(['hangar', 'spigot', 'bukkit', 'modrinth']);
  });
});
