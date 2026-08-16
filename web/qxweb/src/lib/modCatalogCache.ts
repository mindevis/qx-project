import {
  api,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { createTtlCache } from '@/lib/ttlCache';

const TEN_MINUTES = 10 * 60 * 1000;
const FIVE_MINUTES = 5 * 60 * 1000;

const projectCache = createTtlCache<Awaited<ReturnType<typeof api.getModProject>>>(TEN_MINUTES);
const versionListCache = createTtlCache<ModVersion[]>(FIVE_MINUTES);
const versionDetailCache = createTtlCache<ModVersion>(FIVE_MINUTES);

function versionListKey(
  source: ModSource,
  projectId: string,
  loader: string | undefined,
  mcVersion: string,
) {
  return `${source}:${projectId}:${loader ?? ''}:${mcVersion}`;
}

function versionDetailKey(
  source: ModSource,
  projectId: string,
  versionId: string,
  loader: string | undefined,
  mcVersion: string,
) {
  return `${source}:${projectId}:${versionId}:${loader ?? ''}:${mcVersion}`;
}

export function cachedGetModProject(source: ModSource, projectId: string) {
  const key = `${source}:${projectId}`;
  return projectCache.getOrLoad(key, () => api.getModProject(source, projectId));
}

export function cachedListModVersions(
  source: ModSource,
  projectId: string,
  params?: { loader?: string; mc_version?: string },
) {
  const key = versionListKey(source, projectId, params?.loader, params?.mc_version ?? '');
  return versionListCache.getOrLoad(
    key,
    async () => {
      const res = await api.listModVersions(source, projectId, params);
      return res.items ?? [];
    },
    (items) => items.length > 0,
  );
}

export function cachedGetModVersion(
  source: ModSource,
  projectId: string,
  versionId: string,
  params?: { loader?: string; mc_version?: string },
) {
  const key = versionDetailKey(source, projectId, versionId, params?.loader, params?.mc_version ?? '');
  return versionDetailCache.getOrLoad(key, () => api.getModVersion(source, projectId, versionId, params));
}

export function clearModVersionListCache() {
  versionListCache.clear();
  versionDetailCache.clear();
}

export function clearModCatalogCaches() {
  projectCache.clear();
  clearModVersionListCache();
}
