export const CONFIG_EXTENSIONS = ['.toml', '.json', '.properties', '.cfg', '.yml', '.yaml'] as const;

export type ModConfigMod = {
  key: string;
  label: string;
  filename?: string;
  project_name?: string;
  icon_url?: string;
};

export type ModConfigFileEntry = {
  path: string;
  size?: number;
};

export type ModConfigGroup = {
  mod: ModConfigMod;
  files: ModConfigFileEntry[];
};

export type GroupConfigResult = {
  groups: ModConfigGroup[];
  other: ModConfigFileEntry[];
};

type ListDirEntry = {
  path: string;
  dir: boolean;
  name: string;
  size?: number;
};

function normalizeConfigKey(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '');
}

function jarBaseName(filename: string): string {
  let base = filename.replace(/\.jar$/i, '');
  base = base.replace(/[-_+]?\d+(\.\d+)*([+_]\d+(\.\d+)*)?$/i, '');
  return normalizeConfigKey(base);
}

function modIdentifiers(mod: ModConfigMod): string[] {
  const ids = new Set<string>();
  if (mod.filename) {
    ids.add(jarBaseName(mod.filename));
    ids.add(normalizeConfigKey(mod.filename.replace(/\.jar$/i, '')));
  }
  if (mod.project_name) {
    ids.add(normalizeConfigKey(mod.project_name));
    for (const word of mod.project_name.split(/[\s_-]+/)) {
      if (word.length >= 3) {
        ids.add(normalizeConfigKey(word));
      }
    }
  }
  if (mod.key) {
    ids.add(normalizeConfigKey(mod.key));
  }
  return [...ids].filter(Boolean);
}

function configPathKeys(path: string): string[] {
  const relative = path.replace(/^config\/?/i, '');
  const parts = relative.split('/').filter(Boolean);
  const keys = new Set<string>();
  if (parts[0]) {
    keys.add(normalizeConfigKey(parts[0]));
  }
  const basename = parts[parts.length - 1] ?? relative;
  keys.add(normalizeConfigKey(basename.replace(/\.[^.]+$/, '')));
  return [...keys].filter(Boolean);
}

function matchesMod(path: string, mod: ModConfigMod): boolean {
  const pathKeys = configPathKeys(path);
  const modIds = modIdentifiers(mod);
  for (const pathKey of pathKeys) {
    if (!pathKey) continue;
    for (const id of modIds) {
      if (!id) continue;
      if (pathKey === id || pathKey.includes(id) || id.includes(pathKey)) {
        return true;
      }
    }
  }
  return false;
}

export function isConfigFilePath(path: string): boolean {
  const lower = path.toLowerCase();
  return CONFIG_EXTENSIONS.some((ext) => lower.endsWith(ext));
}

export function configRelativePath(path: string): string {
  return path.replace(/^config\/?/i, '');
}

export function filterConfigFileEntries(
  entries: ModConfigFileEntry[],
  query: string,
): ModConfigFileEntry[] {
  const trimmed = query.trim().toLowerCase();
  if (!trimmed) return entries;
  return entries.filter((entry) => {
    const relative = configRelativePath(entry.path).toLowerCase();
    const name = (entry.path.split('/').pop() ?? entry.path).toLowerCase();
    return relative.includes(trimmed) || name.includes(trimmed);
  });
}

export function groupConfigFilesByMod(
  mods: ModConfigMod[],
  configFiles: ModConfigFileEntry[],
): GroupConfigResult {
  const configOnly = configFiles.filter((entry) => isConfigFilePath(entry.path));
  const assigned = new Set<string>();
  const groups: ModConfigGroup[] = [];

  for (const mod of mods) {
    const files = configOnly.filter((entry) => !assigned.has(entry.path) && matchesMod(entry.path, mod));
    for (const entry of files) {
      assigned.add(entry.path);
    }
    if (files.length > 0) {
      groups.push({ mod, files });
    }
  }

  const other = configOnly.filter((entry) => !assigned.has(entry.path));
  return { groups, other };
}

export async function listConfigPaths(
  listDirFn: (path: string) => Promise<ListDirEntry[]>,
  maxDepth = 3,
): Promise<ModConfigFileEntry[]> {
  const files: ModConfigFileEntry[] = [];

  const walk = async (path: string, depth: number) => {
    let entries: ListDirEntry[] = [];
    try {
      entries = await listDirFn(path);
    } catch {
      return;
    }
    for (const entry of entries) {
      if (entry.dir) {
        if (depth < maxDepth) {
          await walk(entry.path, depth + 1);
        }
      } else if (isConfigFilePath(entry.path)) {
        files.push({ path: entry.path, size: entry.size });
      }
    }
  };

  await walk('config', 0);
  return files.sort((a, b) => a.path.localeCompare(b.path));
}
