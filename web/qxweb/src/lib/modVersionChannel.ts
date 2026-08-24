import type { ModSource } from '@/api/client';

export type ModVersionChannel = 'release' | 'beta' | 'alpha';

const EXPLICIT: Record<string, ModVersionChannel> = {
  alpha: 'alpha',
  a: 'alpha',
  beta: 'beta',
  b: 'beta',
  rc: 'beta',
  pre: 'beta',
  prerelease: 'beta',
  preview: 'beta',
  snapshot: 'beta',
  release: 'release',
  stable: 'release',
  ga: 'release',
  final: 'release',
};

export function normalizeModVersionChannel(raw?: string): ModVersionChannel | undefined {
  if (!raw) return undefined;
  return EXPLICIT[raw.trim().toLowerCase()];
}

export function inferModVersionChannel(...names: Array<string | undefined>): ModVersionChannel {
  for (const name of names) {
    const explicit = normalizeModVersionChannel(name);
    if (explicit) return explicit;
    const lower = (name ?? '').toLowerCase();
    if (!lower) continue;
    if (lower.includes('alpha')) return 'alpha';
    if (
      lower.includes('beta') ||
      lower.includes('snapshot') ||
      lower.includes('preview') ||
      lower.includes('-pre') ||
      lower.includes('-rc')
    ) {
      return 'beta';
    }
  }
  return 'release';
}

export function resolveModVersionChannel(
  versionType?: string,
  ...names: Array<string | undefined>
): ModVersionChannel {
  return normalizeModVersionChannel(versionType) ?? inferModVersionChannel(...names);
}

export function catalogProjectUrl(source: ModSource, projectId: string, slug?: string): string {
  const key = (slug || projectId).trim();
  if (!key) return '';
  switch (source) {
    case 'modrinth':
      return `https://modrinth.com/project/${encodeURIComponent(key)}`;
    case 'curseforge':
      return /^\d+$/.test(key)
        ? `https://www.curseforge.com/projects/${key}`
        : `https://www.curseforge.com/minecraft/mc-mods/${encodeURIComponent(key)}`;
    case 'hangar':
      return `https://hangar.papermc.io/${key}`;
    case 'spigot':
      return `https://www.spigotmc.org/resources/${encodeURIComponent(key)}/`;
    case 'bukkit':
      return `https://dev.bukkit.org/projects/${encodeURIComponent(key)}`;
    default:
      return '';
  }
}
