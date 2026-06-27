import type { VpsGameServerType } from '@/lib/gameServerTypes';

export const LAUNCHER_LOADERS = [
  'vanilla',
  'forge',
  'neoforge',
  'fabric',
  'quilt',
] as const;

export type LauncherLoader = (typeof LAUNCHER_LOADERS)[number];

export const DEFAULT_LAUNCHER_LOADER: LauncherLoader = 'vanilla';

export function isLauncherLoader(value: string): value is LauncherLoader {
  return (LAUNCHER_LOADERS as readonly string[]).includes(value);
}

export function launcherLoaderNeedsVersion(loader: LauncherLoader): boolean {
  return loader !== 'vanilla';
}

export function launcherLoaderAsGameServerType(loader: LauncherLoader): VpsGameServerType {
  return loader;
}
