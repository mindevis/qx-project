import {
  api,
  type ModCatalogItem,
  type ModProjectType,
  type ModSource,
} from '@/api/client';
import {
  catalogItemsMatch,
  catalogPartnerSource,
  mergeCatalogCardsByName,
} from '@/lib/mergeCatalogCards';
import { createTtlCache } from '@/lib/ttlCache';

const partnerCache = createTtlCache<ModCatalogItem | null>(5 * 60 * 1000);
const MAX_PARTNER_LOOKUPS = 12;
const PARTNER_CONCURRENCY = 3;

export type CatalogPartnerParams = {
  loader?: string;
  mcVersion: string;
  type?: ModProjectType;
};

function itemKey(item: ModCatalogItem): string {
  return `${item.source}:${item.id}`;
}

async function lookupCatalogPartner(
  item: ModCatalogItem,
  params: CatalogPartnerParams,
): Promise<ModCatalogItem | null> {
  const other = catalogPartnerSource(item.source);
  if (!other) return null;
  const cacheKey = `${other}:${item.slug}:${item.name}:${params.loader ?? ''}:${params.mcVersion}:${params.type ?? ''}`;
  return partnerCache.getOrLoad(cacheKey, async () => {
    const query = item.slug?.trim() || item.name;
    if (!query) return null;
    const res = await api.searchMods({
      q: query,
      type: params.type,
      loader: params.loader,
      mc_version: params.mcVersion,
      source: other,
      limit: 8,
    });
    return (res.items ?? []).find((candidate) => catalogItemsMatch(item, candidate)) ?? null;
  });
}

async function mapPool<T>(items: T[], concurrency: number, worker: (item: T) => Promise<void>) {
  let next = 0;
  const runners = Array.from({ length: Math.min(concurrency, items.length) }, async () => {
    while (next < items.length) {
      const index = next;
      next += 1;
      await worker(items[index]);
    }
  });
  await Promise.all(runners);
}

export async function attachCatalogPartners(
  items: ModCatalogItem[],
  params: CatalogPartnerParams,
): Promise<ModCatalogItem[]> {
  const known = new Set(items.map(itemKey));
  const unpaired = mergeCatalogCardsByName(items, 'all')
    .filter((card) => card.items.length === 1)
    .map((card) => card.items[0])
    .filter((item) => catalogPartnerSource(item.source as ModSource))
    .slice(0, MAX_PARTNER_LOOKUPS);

  const extras: ModCatalogItem[] = [];
  await mapPool(unpaired, PARTNER_CONCURRENCY, async (item) => {
    try {
      const partner = await lookupCatalogPartner(item, params);
      if (partner && !known.has(itemKey(partner))) {
        known.add(itemKey(partner));
        extras.push(partner);
      }
    } catch {
      /* keep the original listing */
    }
  });
  return extras.length === 0 ? items : [...items, ...extras];
}

export function clearCatalogPartnerCache() {
  partnerCache.clear();
}
