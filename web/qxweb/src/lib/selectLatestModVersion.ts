import type { ModVersion } from '@/api/client';

const MOD_PLATFORM_LOADERS = new Set([
  'fabric',
  'forge',
  'neoforge',
  'quilt',
  'bukkit',
  'spigot',
  'paper',
  'purpur',
  'folia',
  'velocity',
  'waterfall',
  'bungeecord',
  'sponge',
]);

function loadersOf(version: ModVersion): string[] {
  return (version.loaders ?? []).map((item) => item.toLowerCase().trim()).filter(Boolean);
}

function primaryFileExt(version: ModVersion): string {
  const name = version.files[0]?.filename ?? '';
  const dot = name.lastIndexOf('.');
  return dot >= 0 ? name.slice(dot).toLowerCase() : '';
}

function hasModPlatformLoader(loaders: string[]): boolean {
  return loaders.some((item) => MOD_PLATFORM_LOADERS.has(item));
}

export function versionMatchesCatalogLoader(version: ModVersion, loader?: string): boolean {
  if (!loader) {
    return true;
  }
  const wanted = loader.toLowerCase().trim();
  const loaders = loadersOf(version);
  if (wanted === 'datapack') {
    if (loaders.includes('datapack') || loaders.includes('data pack')) {
      return true;
    }
    if (hasModPlatformLoader(loaders)) {
      return false;
    }
    return primaryFileExt(version) === '.zip';
  }
  if (loaders.includes('datapack') && !hasModPlatformLoader(loaders)) {
    return false;
  }
  if (loaders.length === 0) {
    return true;
  }
  return loaders.includes(wanted);
}

export function filterVersionsForCatalogLoader(
  versions: ModVersion[],
  loader?: string,
): ModVersion[] {
  if (!loader) {
    return versions;
  }
  return versions.filter((version) => versionMatchesCatalogLoader(version, loader));
}

function matchesMcVersion(version: ModVersion, mcVersion?: string): boolean {
  if (!mcVersion || !version.game_versions?.length) {
    return true;
  }
  return version.game_versions.includes(mcVersion);
}

function publishedAtMs(version: ModVersion): number {
  if (!version.published_at) {
    return 0;
  }
  const parsed = Date.parse(version.published_at);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function selectLatestCompatibleVersion(
  versions: ModVersion[],
  loader?: string,
  mcVersion?: string,
): ModVersion | undefined {
  const byLoader = filterVersionsForCatalogLoader(versions, loader);
  const matching = byLoader.filter((version) => matchesMcVersion(version, mcVersion));
  const pool = matching.length > 0 ? matching : byLoader;
  return [...pool].sort((a, b) => publishedAtMs(b) - publishedAtMs(a))[0];
}
