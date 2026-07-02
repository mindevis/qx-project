import type { GameServerFileEntry, InstanceResource } from '@/api/client';
import type { ModCatalogItem, ModVersion } from '@/api/client';

export type ModSyncSide = 'client' | 'server' | 'both' | 'unknown';

export function modSyncSide(item: Pick<ModCatalogItem, 'client_side' | 'server_side'>): ModSyncSide {
  const client = item.client_side ?? 'unknown';
  const server = item.server_side ?? 'unknown';
  const clientOk = client === 'required' || client === 'optional';
  const serverOk = server === 'required' || server === 'optional';
  if (clientOk && serverOk) return 'both';
  if (serverOk) return 'server';
  if (clientOk) return 'client';
  return 'unknown';
}

export function modSupportsServerSync(item: Pick<ModCatalogItem, 'client_side' | 'server_side'>): boolean {
  const side = modSyncSide(item);
  return side === 'server' || side === 'both' || side === 'unknown';
}

export function isFilenameOnServer(serverFiles: GameServerFileEntry[], filename: string): boolean {
  const lower = filename.toLowerCase();
  return serverFiles.some((entry) => !entry.dir && entry.name.toLowerCase() === lower);
}

export function isModOnServer(
  serverMods: GameServerFileEntry[],
  version: Pick<ModVersion, 'files'>,
): boolean {
  const filenames = version.files.map((f) => f.filename.toLowerCase());
  return serverMods.some((entry) => !entry.dir && filenames.includes(entry.name.toLowerCase()));
}

export function instanceResourceVersionKey(
  resource: Pick<InstanceResource, 'source' | 'project_id' | 'version_id'>,
): string | undefined {
  if (!resource.project_id || !resource.version_id) return undefined;
  return `${resource.source}:${resource.project_id}:${resource.version_id}`;
}

export function isInstanceResourceOnServer(
  serverFiles: GameServerFileEntry[],
  resource: Pick<InstanceResource, 'filename'>,
  versionFilenames?: string[],
): boolean {
  if (versionFilenames?.length) {
    return isModOnServer(serverFiles, {
      files: versionFilenames.map((filename) => ({ filename, url: '' })),
    });
  }
  return isFilenameOnServer(serverFiles, resource.filename);
}

export function instanceResourceSupportsServerSync(resource: InstanceResource): boolean {
  return resource.resource_type === 'mod' && Boolean(resource.project_id && resource.version_id);
}
