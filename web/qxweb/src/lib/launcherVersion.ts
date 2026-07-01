/** Normalize git-describe style versions for comparison (strip leading "v"). */
export function normalizeVersion(value: string): string {
  return value.trim().replace(/^v/i, '');
}

/** Split a version string into comparable numeric segments. */
export function versionParts(value: string): number[] {
  const normalized = normalizeVersion(value);
  const main = normalized.split(/[-+]/, 1)[0] ?? normalized;
  return main
    .split(/[._]/)
    .map((part) => {
      const match = part.match(/^\d+/);
      return match ? Number.parseInt(match[0], 10) : 0;
    })
    .filter((n) => Number.isFinite(n));
}

/** Negative if a < b, positive if a > b, zero if equal or incomparable. */
export function compareVersions(a: string, b: string): number {
  const left = versionParts(a);
  const right = versionParts(b);
  const len = Math.max(left.length, right.length);
  for (let i = 0; i < len; i += 1) {
    const diff = (left[i] ?? 0) - (right[i] ?? 0);
    if (diff !== 0) return diff;
  }
  return normalizeVersion(a).localeCompare(normalizeVersion(b));
}

export function isUpdateAvailable(installed: string | undefined, latest: string): boolean {
  const latestTrimmed = latest.trim();
  if (!latestTrimmed) return false;
  const installedTrimmed = installed?.trim();
  if (!installedTrimmed) return true;
  return compareVersions(installedTrimmed, latestTrimmed) < 0;
}
