import type { McVersionItem } from '@/launcher/mcVersions';
import type { VpsGameServerType } from '@/lib/gameServerTypes';
import { gameServerUpstreamUrl } from '@/lib/gameServerUpstream';

export type VersionOption = {
  value: string;
  label: string;
};

const PAPER_API = gameServerUpstreamUrl('papermc', '/v2/projects');
const PURPUR_API = gameServerUpstreamUrl('purpur', '/v2/projects/purpur');
const FORGE_PROMOTIONS = gameServerUpstreamUrl(
  'forge',
  '/net/minecraftforge/forge/promotions_slim.json',
);
const FORGE_MAVEN_METADATA = gameServerUpstreamUrl(
  'mavenforge',
  '/net/minecraftforge/forge/maven-metadata.xml',
);
const NEOFORGE_VERSIONS = gameServerUpstreamUrl(
  'neoforge',
  '/api/maven/versions/releases/net/neoforged/neoforge',
);
const FABRIC_GAME_VERSIONS = gameServerUpstreamUrl('fabric', '/v2/versions/game');
const FABRIC_LOADER_VERSIONS = gameServerUpstreamUrl('fabric', '/v2/versions/loader');
const QUILT_GAME_VERSIONS = gameServerUpstreamUrl('quilt', '/v3/versions/game');
const QUILT_LOADER_VERSIONS = gameServerUpstreamUrl('quilt', '/v3/versions/loader');
const MOHIST_PROJECT = gameServerUpstreamUrl('mohist', '/api/v2/projects/mohist');
const MAGMA_VERSIONS = gameServerUpstreamUrl('magma', '/api/versions?limit=0');
const ARCLIGHT_MINECRAFT = gameServerUpstreamUrl('arclight', '/v1/files/arclight/minecraft');

const FORGE_ARTIFACT_VERSION = /^(\d+\.\d+(?:\.\d+)?)-(.+)$/;
const LOADER_COMPAT_BATCH_SIZE = 15;

let forgePromosCache: Promise<Record<string, string>> | null = null;
let forgeMavenVersionsCache: Promise<string[]> | null = null;

const upstreamJsonCache = new Map<string, { expiresAt: number; promise: Promise<unknown> }>();
const UPSTREAM_JSON_TTL_MS = 10 * 60 * 1000;

async function cachedFetchJson<T>(url: string): Promise<T> {
  const now = Date.now();
  const hit = upstreamJsonCache.get(url);
  if (hit && now < hit.expiresAt) {
    return hit.promise as Promise<T>;
  }
  const promise = fetchJson<T>(url).catch((error) => {
    upstreamJsonCache.delete(url);
    throw error;
  });
  upstreamJsonCache.set(url, { expiresAt: now + UPSTREAM_JSON_TTL_MS, promise });
  return promise;
}

export function clearGameServerVersionsCache() {
  upstreamJsonCache.clear();
  forgePromosCache = null;
  forgeMavenVersionsCache = null;
}

function compareMcVersionsDesc(a: string, b: string): number {
  const pa = a.split('.').map((part) => Number.parseInt(part, 10) || 0);
  const pb = b.split('.').map((part) => Number.parseInt(part, 10) || 0);
  const len = Math.max(pa.length, pb.length);
  for (let i = 0; i < len; i += 1) {
    const diff = (pb[i] ?? 0) - (pa[i] ?? 0);
    if (diff !== 0) return diff;
  }
  return 0;
}

function compareVersionsDesc(a: string, b: string): number {
  return b.localeCompare(a, undefined, { numeric: true });
}

export function gameServerTypeNeedsLoader(serverType: VpsGameServerType): boolean {
  return serverType !== 'vanilla';
}

export function mcVersionOptionsFromItems(items: McVersionItem[]): VersionOption[] {
  return items
    .filter((item) => item.type === 'release')
    .map((item) => ({ value: item.id, label: item.id }))
    .sort((a, b) => compareMcVersionsDesc(a.value, b.value));
}

async function fetchJson<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`upstream ${res.status}`);
  }
  return (await res.json()) as T;
}

async function fetchText(url: string): Promise<string> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`upstream ${res.status}`);
  }
  return res.text();
}

async function fetchHeadOk(url: string): Promise<boolean> {
  const res = await fetch(url, { method: 'HEAD' });
  return res.ok;
}

function parseForgeArtifactVersion(full: string): { mc: string; loader: string } | null {
  const match = FORGE_ARTIFACT_VERSION.exec(full);
  if (!match) return null;
  return { mc: match[1], loader: match[2] };
}

function parseMavenMetadataVersions(xml: string): string[] {
  const versions: string[] = [];
  const re = /<version>([^<]+)<\/version>/g;
  let match = re.exec(xml);
  while (match) {
    versions.push(match[1]);
    match = re.exec(xml);
  }
  return versions;
}

async function fetchForgeMavenVersions(): Promise<string[]> {
  if (!forgeMavenVersionsCache) {
    forgeMavenVersionsCache = fetchText(FORGE_MAVEN_METADATA)
      .then(parseMavenMetadataVersions)
      .catch((error) => {
        forgeMavenVersionsCache = null;
        throw error;
      });
  }
  return forgeMavenVersionsCache;
}

async function fetchForgePromos(): Promise<Record<string, string>> {
  if (!forgePromosCache) {
    forgePromosCache = fetchJson<{ promos: Record<string, string> }>(FORGE_PROMOTIONS)
      .then((data) => data.promos)
      .catch((error) => {
        forgePromosCache = null;
        throw error;
      });
  }
  return forgePromosCache;
}

async function fetchPaperFamilyMcVersions(project: 'paper' | 'purpur'): Promise<VersionOption[]> {
  const base = project === 'paper' ? `${PAPER_API}/paper` : PURPUR_API;
  const data = await cachedFetchJson<{ versions: string[] }>(base);
  return [...data.versions]
    .sort(compareMcVersionsDesc)
    .map((version) => ({ value: version, label: version }));
}

async function fetchPaperFamilyBuilds(
  project: 'paper' | 'purpur',
  mcVersion: string,
): Promise<VersionOption[]> {
  const base =
    project === 'paper'
      ? `${PAPER_API}/paper/versions/${mcVersion}/builds`
      : `${PURPUR_API}/versions/${mcVersion}/builds`;
  const data = await cachedFetchJson<{ builds: { build: number }[] }>(base);
  return [...data.builds]
    .sort((a, b) => b.build - a.build)
    .map((item) => ({
      value: String(item.build),
      label: `#${item.build}`,
    }));
}

async function fetchForgeMcVersions(): Promise<VersionOption[]> {
  const versions = await fetchForgeMavenVersions();
  const mcVersions = new Set<string>();
  for (const full of versions) {
    const parsed = parseForgeArtifactVersion(full);
    if (parsed) mcVersions.add(parsed.mc);
  }
  return [...mcVersions]
    .sort(compareMcVersionsDesc)
    .map((version) => ({ value: version, label: version }));
}

async function fetchForgeLoaderVersions(mcVersion: string): Promise<VersionOption[]> {
  const [mavenVersions, promos] = await Promise.all([
    fetchForgeMavenVersions(),
    fetchForgePromos(),
  ]);
  const recommended = promos[`${mcVersion}-recommended`];
  const latest = promos[`${mcVersion}-latest`];
  const loaders = new Set<string>();

  for (const full of mavenVersions) {
    const parsed = parseForgeArtifactVersion(full);
    if (parsed?.mc === mcVersion) {
      loaders.add(parsed.loader);
    }
  }

  return [...loaders]
    .sort(compareVersionsDesc)
    .map((loader) => {
      let label = loader;
      if (loader === recommended) {
        label = `${loader} (recommended)`;
      } else if (loader === latest) {
        label = `${loader} (latest)`;
      }
      return { value: loader, label };
    });
}

function neoForgeVersionPrefix(mcVersion: string): string | null {
  const parts = mcVersion.split('.');
  if (parts.length < 2 || parts[0] !== '1') return null;
  if (parts.length === 2) {
    return `${parts[1]}.0.`;
  }
  return `${parts[1]}.${parts[2]}.`;
}

function isPreReleaseNeoForgeVersion(version: string): boolean {
  return version.includes('beta') || version.includes('alpha');
}

function neoForgeMcVersion(neoforgeVersion: string): string | null {
  const parts = neoforgeVersion.split('.');
  if (parts.length < 2) return null;
  const major = Number.parseInt(parts[0] ?? '', 10);
  if (Number.isNaN(major)) return null;
  if (major === 21) {
    if (parts[1] === '0') return '1.21';
    return `1.21.${parts[1]}`;
  }
  if (major === 20) {
    return `1.20.${parts[1]}`;
  }
  return null;
}

/** Maven JSON API returns `{ versions: string[] }`; tests and older mirrors may return a raw array. */
function parseNeoForgeVersionsPayload(data: unknown): string[] {
  if (Array.isArray(data)) {
    return data.filter((item): item is string => typeof item === 'string');
  }
  if (data && typeof data === 'object' && 'versions' in data) {
    const versions = (data as { versions: unknown }).versions;
    if (Array.isArray(versions)) {
      return versions.filter((item): item is string => typeof item === 'string');
    }
  }
  throw new Error('unexpected neoforge versions payload');
}

async function fetchNeoForgeVersionList(): Promise<string[]> {
  const data = await cachedFetchJson<unknown>(NEOFORGE_VERSIONS);
  return parseNeoForgeVersionsPayload(data);
}

async function fetchNeoForgeMcVersions(): Promise<VersionOption[]> {
  const versions = await fetchNeoForgeVersionList();
  const mcVersions = new Set<string>();
  for (const loaderVersion of versions) {
    if (isPreReleaseNeoForgeVersion(loaderVersion)) continue;
    const mcVersion = neoForgeMcVersion(loaderVersion);
    if (mcVersion) mcVersions.add(mcVersion);
  }
  return [...mcVersions]
    .sort(compareMcVersionsDesc)
    .map((version) => ({ value: version, label: version }));
}

async function fetchNeoForgeLoaderVersions(mcVersion: string): Promise<VersionOption[]> {
  const prefix = neoForgeVersionPrefix(mcVersion);
  if (!prefix) return [];
  const versions = await fetchNeoForgeVersionList();
  return versions
    .filter(
      (loaderVersion) =>
        !isPreReleaseNeoForgeVersion(loaderVersion) && loaderVersion.startsWith(prefix),
    )
    .sort(compareVersionsDesc)
    .map((version) => ({ value: version, label: version }));
}

async function fetchStableGameVersions(url: string): Promise<VersionOption[]> {
  const items = await cachedFetchJson<{ version: string; stable: boolean }[]>(url);
  return items
    .filter((item) => item.stable)
    .map((item) => ({ value: item.version, label: item.version }))
    .sort((a, b) => compareMcVersionsDesc(a.value, b.value));
}

async function fetchCompatibleLoaderVersions(
  loaderListUrl: string,
  checkUrl: (loaderVersion: string, mcVersion: string) => string,
  mcVersion: string,
): Promise<VersionOption[]> {
  const loaders = await cachedFetchJson<{ version: string }[]>(loaderListUrl);
  const compatible: VersionOption[] = [];

  for (let i = 0; i < loaders.length; i += LOADER_COMPAT_BATCH_SIZE) {
    const batch = loaders.slice(i, i + LOADER_COMPAT_BATCH_SIZE);
    const results = await Promise.all(
      batch.map(async (loader) => {
        const ok = await fetchHeadOk(checkUrl(loader.version, mcVersion));
        return ok ? loader.version : null;
      }),
    );
    for (const version of results) {
      if (version) {
        compatible.push({ value: version, label: version });
      }
    }
  }

  return compatible.sort((a, b) => compareVersionsDesc(a.value, b.value));
}

type ArclightFileEntry = { name: string; type: string };

async function fetchArclightFileNames(url: string): Promise<string[]> {
  const data = await cachedFetchJson<{ files: ArclightFileEntry[] }>(url);
  return data.files.map((file) => file.name);
}

async function fetchMohistMcVersions(): Promise<VersionOption[]> {
  const data = await cachedFetchJson<{ versions: string[] }>(MOHIST_PROJECT);
  return [...data.versions]
    .sort(compareMcVersionsDesc)
    .map((version) => ({ value: version, label: version }));
}

async function fetchMohistLoaderVersions(mcVersion: string): Promise<VersionOption[]> {
  const data = await cachedFetchJson<{ builds: { number: number }[] }>(
    `${MOHIST_PROJECT}/${mcVersion}/builds`,
  );
  return [...data.builds]
    .sort((a, b) => b.number - a.number)
    .map((build) => ({
      value: String(build.number),
      label: `#${build.number}`,
    }));
}

type MagmaVersionEntry = {
  version: string;
  minecraftVersion: string;
};

function magmaMcMatches(apiMc: string, selectedMc: string): boolean {
  if (apiMc === selectedMc) return true;
  if (apiMc.endsWith('.x')) {
    const prefix = apiMc.slice(0, -2);
    return selectedMc === prefix || selectedMc.startsWith(`${prefix}.`);
  }
  return selectedMc.startsWith(`${apiMc}.`) || apiMc.startsWith(`${selectedMc}.`);
}

async function fetchMagmaVersions(): Promise<MagmaVersionEntry[]> {
  const data = await cachedFetchJson<{ versions: MagmaVersionEntry[] }>(MAGMA_VERSIONS);
  return data.versions;
}

async function fetchMagmaMcVersions(fallbackMcVersions: McVersionItem[]): Promise<VersionOption[]> {
  const versions = await fetchMagmaVersions();
  const patterns = new Set(versions.map((item) => item.minecraftVersion));
  const options = new Map<string, VersionOption>();

  for (const item of mcVersionOptionsFromItems(fallbackMcVersions)) {
    if ([...patterns].some((pattern) => magmaMcMatches(pattern, item.value))) {
      options.set(item.value, item);
    }
  }

  for (const pattern of patterns) {
    if (!pattern.endsWith('.x')) {
      options.set(pattern, { value: pattern, label: pattern });
    }
  }

  return [...options.values()].sort((a, b) => compareMcVersionsDesc(a.value, b.value));
}

async function fetchMagmaLoaderVersions(mcVersion: string): Promise<VersionOption[]> {
  const versions = await fetchMagmaVersions();
  return versions
    .filter((item) => magmaMcMatches(item.minecraftVersion, mcVersion))
    .sort((a, b) => compareVersionsDesc(a.version, b.version))
    .map((item) => ({
      value: item.version,
      label: item.version,
    }));
}

async function fetchArclightMcVersions(): Promise<VersionOption[]> {
  const names = await fetchArclightFileNames(ARCLIGHT_MINECRAFT);
  return names
    .sort(compareMcVersionsDesc)
    .map((version) => ({ value: version, label: version }));
}

async function fetchArclightLoaderVersions(mcVersion: string): Promise<VersionOption[]> {
  const loaders = await fetchArclightFileNames(
    gameServerUpstreamUrl('arclight', `/v1/files/arclight/minecraft/${mcVersion}/loaders`),
  );
  const options = new Map<string, VersionOption>();

  for (const loader of loaders) {
    const stableUrl = gameServerUpstreamUrl(
      'arclight',
      `/v1/files/arclight/minecraft/${mcVersion}/loaders/${loader}/versions-stable`,
    );
    const snapshotUrl = gameServerUpstreamUrl(
      'arclight',
      `/v1/files/arclight/minecraft/${mcVersion}/loaders/${loader}/versions-snapshot`,
    );

    const [stable, snapshot] = await Promise.all([
      fetchArclightFileNames(stableUrl).catch(() => [] as string[]),
      fetchArclightFileNames(snapshotUrl).catch(() => [] as string[]),
    ]);

    for (const version of [...stable, ...snapshot]) {
      const value = `${loader}:${version}`;
      options.set(value, { value, label: `${loader} ${version}` });
    }
  }

  return [...options.values()].sort((a, b) => compareVersionsDesc(a.value, b.value));
}

function fallbackMcOptions(items: McVersionItem[]): VersionOption[] {
  return mcVersionOptionsFromItems(items);
}

export async function listGameServerMcVersions(
  serverType: VpsGameServerType,
  fallbackMcVersions: McVersionItem[],
): Promise<VersionOption[]> {
  try {
    switch (serverType) {
      case 'vanilla':
        return mcVersionOptionsFromItems(fallbackMcVersions);
      case 'paper':
      case 'spigot':
        return await fetchPaperFamilyMcVersions('paper');
      case 'purpur':
        return await fetchPaperFamilyMcVersions('purpur');
      case 'forge':
        return await fetchForgeMcVersions();
      case 'neoforge':
        return await fetchNeoForgeMcVersions();
      case 'fabric':
        return await fetchStableGameVersions(FABRIC_GAME_VERSIONS);
      case 'quilt':
        return await fetchStableGameVersions(QUILT_GAME_VERSIONS);
      case 'mohist':
        return await fetchMohistMcVersions();
      case 'magma':
        return await fetchMagmaMcVersions(fallbackMcVersions);
      case 'arclight':
        return await fetchArclightMcVersions();
      default:
        return mcVersionOptionsFromItems(fallbackMcVersions);
    }
  } catch {
    return fallbackMcOptions(fallbackMcVersions);
  }
}

export async function listGameServerLoaderVersions(
  serverType: VpsGameServerType,
  mcVersion: string,
): Promise<VersionOption[]> {
  if (!gameServerTypeNeedsLoader(serverType) || !mcVersion) {
    return [];
  }
  try {
    switch (serverType) {
      case 'paper':
        return await fetchPaperFamilyBuilds('paper', mcVersion);
      case 'purpur':
        return await fetchPaperFamilyBuilds('purpur', mcVersion);
      case 'spigot':
        return await fetchPaperFamilyBuilds('paper', mcVersion);
      case 'forge':
        return await fetchForgeLoaderVersions(mcVersion);
      case 'neoforge':
        return await fetchNeoForgeLoaderVersions(mcVersion);
      case 'fabric':
        return await fetchCompatibleLoaderVersions(
          FABRIC_LOADER_VERSIONS,
          (loader, game) =>
            gameServerUpstreamUrl('fabric', `/v2/versions/loader/${game}/${loader}`),
          mcVersion,
        );
      case 'quilt':
        return await fetchCompatibleLoaderVersions(
          QUILT_LOADER_VERSIONS,
          (loader, game) =>
            gameServerUpstreamUrl('quilt', `/v3/versions/loader/${game}/${loader}`),
          mcVersion,
        );
      case 'mohist':
        return await fetchMohistLoaderVersions(mcVersion);
      case 'magma':
        return await fetchMagmaLoaderVersions(mcVersion);
      case 'arclight':
        return await fetchArclightLoaderVersions(mcVersion);
      default:
        return [];
    }
  } catch {
    return [];
  }
}

export function formatGameServerMcVersionLabel(mcVersion?: string): string {
  return mcVersion ?? '—';
}

export function formatGameServerLoaderVersionLabel(
  loaderVersion?: string,
  serverType?: VpsGameServerType,
): string {
  if (!serverType || serverType === 'vanilla') return '—';
  return loaderVersion ?? '—';
}
