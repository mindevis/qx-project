import type { ModCatalogItem, ModCatalogSourceFilter, ModProjectType, ModSource } from '@/api/client';

export type CatalogCard = {
  key: string;
  name: string;
  items: ModCatalogItem[];
};

const SOURCE_ORDER: Record<ModSource, number> = {
  hangar: 0,
  spigot: 1,
  bukkit: 2,
  modrinth: 3,
  curseforge: 4,
  upload: 5,
};

export function catalogItemNameKey(name: string): string {
  return name.trim().replace(/\s+/g, ' ').toLowerCase();
}

export function catalogItemMatchKeys(item: Pick<ModCatalogItem, 'name' | 'slug'>): string[] {
  const keys = new Set<string>();
  const name = catalogItemNameKey(item.name);
  if (name) keys.add(`n:${name}`);
  const stripped = catalogItemNameKey(item.name.replace(/\s*[([][^)\]]*[)\]]/g, ' '));
  if (stripped) keys.add(`n:${stripped}`);
  const slug = item.slug?.trim().toLowerCase();
  if (slug) keys.add(`s:${slug}`);
  return [...keys];
}

export function catalogPartnerSource(source: ModSource): ModSource | null {
  if (source === 'modrinth') return 'curseforge';
  if (source === 'curseforge') return 'modrinth';
  return null;
}

export function catalogPartnerSources(source: ModSource, projectType?: ModProjectType): ModSource[] {
  const pool: ModSource[] =
    projectType === 'plugin' ? ['modrinth', 'hangar', 'spigot', 'bukkit'] : ['modrinth', 'curseforge'];
  return pool.filter((item) => item !== source);
}

export function catalogItemsMatch(left: ModCatalogItem, right: ModCatalogItem): boolean {
  if (left.source === right.source) return false;
  if (left.source === 'upload' || right.source === 'upload') return false;
  const rightKeys = new Set(catalogItemMatchKeys(right));
  return catalogItemMatchKeys(left).some((key) => rightKeys.has(key));
}

export function catalogCardKey(items: ModCatalogItem[]): string {
  return [...items]
    .map((item) => `${item.source}:${item.id}`)
    .sort()
    .join('|');
}

export function preferredCatalogItem(items: ModCatalogItem[]): ModCatalogItem {
  return [...items].sort((a, b) => {
    const downloads = (b.downloads ?? 0) - (a.downloads ?? 0);
    if (downloads !== 0) return downloads;
    return SOURCE_ORDER[a.source] - SOURCE_ORDER[b.source];
  })[0];
}

export function catalogCardItem(card: CatalogCard, source?: ModSource): ModCatalogItem {
  if (source) {
    return card.items.find((item) => item.source === source) ?? preferredCatalogItem(card.items);
  }
  return preferredCatalogItem(card.items);
}

function toCard(item: ModCatalogItem): CatalogCard {
  return {
    key: `${item.source}:${item.id}`,
    name: item.name,
    items: [item],
  };
}

function toMergedGroup(items: ModCatalogItem[]): CatalogCard {
  const sorted = [...items].sort((a, b) => SOURCE_ORDER[a.source] - SOURCE_ORDER[b.source]);
  return {
    key: catalogCardKey(sorted),
    name: items[0].name,
    items: sorted,
  };
}

export function mergeCatalogCardsByName(
  items: ModCatalogItem[],
  sourceFilter: ModCatalogSourceFilter,
): CatalogCard[] {
  if (sourceFilter !== 'all') {
    return items.map(toCard);
  }

  const used = new Set<number>();
  const cards: CatalogCard[] = [];

  for (let i = 0; i < items.length; i++) {
    if (used.has(i)) continue;
    const group = [items[i]];
    used.add(i);
    for (let j = i + 1; j < items.length; j++) {
      if (used.has(j)) continue;
      if (group.some((item) => catalogItemsMatch(item, items[j]))) {
        group.push(items[j]);
        used.add(j);
      }
    }
    cards.push(group.length === 1 ? toCard(group[0]) : toMergedGroup(group));
  }

  return cards;
}
