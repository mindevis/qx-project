import type { LauncherLoader } from '@/lib/launcherLoaders';

const MODDED_LOADERS = new Set<LauncherLoader>(['forge', 'neoforge', 'fabric', 'quilt']);

export function isModdedLauncherLoader(loader: string): loader is LauncherLoader {
  return MODDED_LOADERS.has(loader as LauncherLoader);
}
