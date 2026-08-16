import type { ModCatalogItem, ModCatalogSourceFilter, ModSource } from '@/api/client';

export type CatalogCard = {
  key: string;
  name: string;
  items: ModCatalogItem[];
};

const SOURCE_ORDER: Record<ModSource, number> = {
  modrinth: 0,
  curseforge: 1,
  upload: 2,
};

export function catalogItemNameKey(name: string): string {
  return name.trim().replace(/\s+/g, ' ').toLowerCase();
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

function toMergedCard(first: ModCatalogItem, second: ModCatalogItem): CatalogCard {
  const items = [first, second].sort((a, b) => SOURCE_ORDER[a.source] - SOURCE_ORDER[b.source]);
  return {
    key: catalogCardKey(items),
    name: first.name,
    items,
  };
}

function partnerSource(source: ModSource): ModSource | null {
  if (source === 'modrinth') return 'curseforge';
  if (source === 'curseforge') return 'modrinth';
  return null;
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
    const item = items[i];
    const other = partnerSource(item.source);
    if (!other) {
      used.add(i);
      cards.push(toCard(item));
      continue;
    }

    const nameKey = catalogItemNameKey(item.name);
    const partnerIndex = items.findIndex(
      (candidate, index) =>
        index > i &&
        !used.has(index) &&
        candidate.source === other &&
        catalogItemNameKey(candidate.name) === nameKey,
    );

    used.add(i);
    if (partnerIndex >= 0) {
      used.add(partnerIndex);
      cards.push(toMergedCard(item, items[partnerIndex]));
    } else {
      cards.push(toCard(item));
    }
  }

  return cards;
}
