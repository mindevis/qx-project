/** Fallback when Mojang manifest is unreachable. */
export const FALLBACK_MC_VERSIONS = ['1.20.4', '1.21', '1.21.1'] as const;

export const DEFAULT_MC_VERSION = '1.21';

export const MC_VERSION_TYPE_ORDER = ['release', 'snapshot', 'old_beta', 'old_alpha'] as const;

export type McVersionItem = {
  id: string;
  type: string;
};

export type McVersionsList = {
  latest?: Record<string, string>;
  items: McVersionItem[];
};

export function pickDefaultMcVersion(
  latest?: Record<string, string>,
  items?: McVersionItem[],
): string {
  if (latest?.release) {
    return latest.release;
  }
  const firstRelease = items?.find((item) => item.type === 'release');
  return firstRelease?.id ?? DEFAULT_MC_VERSION;
}

export function groupMcVersionOptions(
  items: McVersionItem[],
  typeLabel: (type: string) => string,
): { label: string; options: { value: string; label: string }[] }[] {
  const byType = new Map<string, McVersionItem[]>();
  for (const item of items) {
    const list = byType.get(item.type) ?? [];
    list.push(item);
    byType.set(item.type, list);
  }

  const groups: { label: string; options: { value: string; label: string }[] }[] = [];
  for (const type of MC_VERSION_TYPE_ORDER) {
    const versions = byType.get(type);
    if (!versions?.length) continue;
    groups.push({
      label: typeLabel(type),
      options: versions.map((version) => ({ value: version.id, label: version.id })),
    });
  }

  for (const [type, versions] of byType) {
    if (MC_VERSION_TYPE_ORDER.includes(type as (typeof MC_VERSION_TYPE_ORDER)[number])) continue;
    groups.push({
      label: typeLabel(type),
      options: versions.map((version) => ({ value: version.id, label: version.id })),
    });
  }

  return groups;
}

export function fallbackMcVersionsList(): McVersionsList {
  return {
    items: FALLBACK_MC_VERSIONS.map((id) => ({ id, type: 'release' })),
  };
}
