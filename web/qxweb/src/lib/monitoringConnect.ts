import type { LauncherInstance, MonitoringServer, UserProfile } from '@/api/client';
import {
  isLauncherLoader,
  launcherLoaderNeedsVersion,
  type LauncherLoader,
} from '@/lib/launcherLoaders';

export type MonitoringLauncherSpec = {
  name: string;
  mc_version: string;
  loader: LauncherLoader;
  loader_version?: string;
};

export function launcherSpecForMonitoringServer(
  server: Pick<MonitoringServer, 'name' | 'server_type' | 'mc_version' | 'loader_version'>,
): MonitoringLauncherSpec | null {
  const mcVersion = server.mc_version.trim();
  if (!mcVersion) return null;

  const name = server.name.trim().slice(0, 64) || 'Minecraft';
  if (isLauncherLoader(server.server_type)) {
    const loader = server.server_type;
    const loaderVersion = server.loader_version?.trim();
    if (launcherLoaderNeedsVersion(loader) && !loaderVersion) {
      return null;
    }
    return {
      name,
      mc_version: mcVersion,
      loader,
      loader_version: loaderVersion || undefined,
    };
  }

  return { name, mc_version: mcVersion, loader: 'vanilla' };
}

export function findCompatibleInstance(
  instances: LauncherInstance[],
  spec: MonitoringLauncherSpec,
): LauncherInstance | undefined {
  return instances.find((item) => {
    if (item.mc_version !== spec.mc_version || item.loader !== spec.loader) {
      return false;
    }
    if (spec.loader_version && item.loader_version !== spec.loader_version) {
      return false;
    }
    return true;
  });
}

export function offlineUsernameFromIdentity(
  user: Pick<UserProfile, 'id' | 'email' | 'username'> | null,
): string {
  const candidates = [user?.username, user?.email?.split('@')[0], 'Player'];
  for (const raw of candidates) {
    const cleaned = (raw ?? '').replace(/[^a-zA-Z0-9_]/g, '').slice(0, 16);
    if (cleaned.length >= 3) return cleaned;
  }
  return 'Player';
}
