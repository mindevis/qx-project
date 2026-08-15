import type { ModVersion } from '@/api/client';

function matchesLoader(version: ModVersion, loader?: string): boolean {
  if (!loader || !version.loaders?.length) {
    return true;
  }
  const wanted = loader.toLowerCase();
  return version.loaders.some((item) => item.toLowerCase() === wanted);
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
  const matching = versions.filter(
    (version) => matchesLoader(version, loader) && matchesMcVersion(version, mcVersion),
  );
  const pool = matching.length > 0 ? matching : versions;
  return [...pool].sort((a, b) => publishedAtMs(b) - publishedAtMs(a))[0];
}
