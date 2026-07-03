import { api, type McVersionsList } from '@/api/client';
import { createTtlCache } from '@/lib/ttlCache';

const ONE_HOUR = 60 * 60 * 1000;

const cache = createTtlCache<McVersionsList>(ONE_HOUR);

export function cachedListMcVersions() {
  return cache.getOrLoad('launcher-mc-versions', () => api.listMcVersions());
}

export function clearMcVersionsCache() {
  cache.clear();
}
