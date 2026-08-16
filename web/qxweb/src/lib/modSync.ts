import {
  api,
  type GameServerContentSyncBody,
  type GameServerFileEntry,
  type InstanceResource,
  type ModCatalogItem,
  type ModSource,
  type ModVersion,
} from '@/api/client';

export type ModSyncSelectionInput = {
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
};

function installedProjectKey(source: string, projectId: string) {
  return `${source}:${projectId}`;
}

export function buildModSyncBodies(
  selection: ModSyncSelectionInput,
  installedResources: InstanceResource[],
  versionDetail: ModVersion,
): GameServerContentSyncBody[] {
  const installedByProject = new Map(
    installedResources
      .filter((resource) => resource.project_id)
      .map((resource) => [installedProjectKey(resource.source, resource.project_id!), resource] as const),
  );

  const bodies: GameServerContentSyncBody[] = [];

  for (const dep of versionDetail.dependencies ?? []) {
    if (dep.dependency_type !== 'required' || !dep.project_id) continue;
    const resource = installedByProject.get(installedProjectKey(dep.source, dep.project_id));
    if (!resource) continue;

    const filename = dep.filename ?? resource.filename;
    const downloadUrl = dep.download_url;
    const versionId = dep.version_id ?? resource.version_id;
    if (!filename || !downloadUrl || !versionId) continue;

    bodies.push({
      source: dep.source,
      project_id: dep.project_id,
      version_id: versionId,
      filename,
      download_url: downloadUrl,
      project_name: dep.project_name ?? resource.project_name,
      version_number: dep.version_number ?? resource.version_number,
    });
  }

  const mainFile = selection.version.files[0];
  if (!mainFile?.url) {
    throw new Error('mod sync file missing');
  }

  bodies.push({
    source: selection.source,
    project_id: selection.projectId,
    version_id: selection.version.id,
    filename: mainFile.filename,
    download_url: mainFile.url,
    project_name: selection.projectName,
    version_number: selection.version.version_number,
  });

  return bodies;
}

export async function resolveModSyncBodies(
  selection: ModSyncSelectionInput,
  installedResources: InstanceResource[],
  loader: string,
  mcVersion?: string,
): Promise<GameServerContentSyncBody[]> {
  const params = { loader, mc_version: mcVersion };
  let versionDetail = selection.version;

  if (!versionDetail.dependencies?.length) {
    try {
      versionDetail = await api.getModVersion(
        selection.source,
        selection.projectId,
        selection.version.id,
        params,
      );
    } catch {
      versionDetail = selection.version;
    }
  }

  const dependencies = [...(versionDetail.dependencies ?? [])];
  const installedByProject = new Map(
    installedResources
      .filter((resource) => resource.project_id)
      .map((resource) => [installedProjectKey(resource.source, resource.project_id!), resource] as const),
  );

  for (let index = 0; index < dependencies.length; index += 1) {
    const dep = dependencies[index];
    if (dep.dependency_type !== 'required' || !dep.project_id || dep.download_url) continue;

    const resource = installedByProject.get(installedProjectKey(dep.source, dep.project_id));
    if (!resource?.project_id || !resource.version_id) continue;

    try {
      const depVersion = await api.getModVersion(
        dep.source,
        resource.project_id,
        resource.version_id,
        params,
      );
      const file = depVersion.files[0];
      if (!file?.url) continue;
      dependencies[index] = {
        ...dep,
        download_url: file.url,
        filename: dep.filename ?? file.filename ?? resource.filename,
        version_id: dep.version_id ?? depVersion.id,
        version_number: dep.version_number ?? depVersion.version_number,
      };
    } catch {
      // Keep dependency unresolved; it will be skipped by buildModSyncBodies.
    }
  }

  return buildModSyncBodies(selection, installedResources, {
    ...versionDetail,
    dependencies,
  });
}

export type ModSyncSide = 'client' | 'server' | 'both' | 'unknown';

export type ModTarget =
  | 'mods'
  | 'client-mods'
  | 'resourcepacks'
  | 'client-resourcepacks'
  | 'shaderpacks'
  | 'client-shaders';

export type ContentTarget = ModTarget;

export function modSyncSide(item: Pick<ModCatalogItem, 'client_side' | 'server_side'>): ModSyncSide {
  const client = item.client_side?.trim() || 'unknown';
  const server = item.server_side?.trim() || 'unknown';
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

export function isCatalogItemOnServer(
  item: Pick<ModCatalogItem, 'id' | 'slug' | 'name'>,
  serverFiles: GameServerFileEntry[],
): boolean {
  const tokens = [item.slug, item.id]
    .filter((value): value is string => Boolean(value && value.length >= 3))
    .map((value) => value.toLowerCase());
  const nameNorm = item.name.toLowerCase().replace(/[^a-z0-9]+/g, '');
  return serverFiles.some((entry) => {
    if (entry.dir) return false;
    const filename = entry.name.toLowerCase();
    const filenameNorm = filename.replace(/[^a-z0-9]+/g, '');
    if (tokens.some((token) => filename.includes(token))) {
      return true;
    }
    return nameNorm.length >= 4 && filenameNorm.includes(nameNorm);
  });
}

export function instanceResourceFileEntries(
  resources: Array<Pick<InstanceResource, 'filename'>>,
): GameServerFileEntry[] {
  return resources
    .filter((resource) => resource.filename)
    .map((resource) => ({
      name: resource.filename,
      path: resource.filename,
      dir: false,
    }));
}

export function isCatalogItemInstalledOnInstance(
  item: Pick<ModCatalogItem, 'id' | 'slug' | 'name' | 'source'>,
  resources: Array<Pick<InstanceResource, 'source' | 'project_id' | 'filename'>>,
): boolean {
  if (
    item.id &&
    resources.some((resource) => resource.project_id === item.id && resource.source === item.source)
  ) {
    return true;
  }
  return isCatalogItemOnServer(item, instanceResourceFileEntries(resources));
}

export function instanceResourceVersionKey(
  resource: Pick<InstanceResource, 'source' | 'project_id' | 'version_id' | 'filename'>,
): string | undefined {
  if (resource.project_id && resource.version_id) {
    return `${resource.source}:${resource.project_id}:${resource.version_id}`;
  }
  if (resource.source === 'upload' && resource.filename) {
    return `upload:${resource.filename}`;
  }
  return undefined;
}

export function isUploadedInstanceResource(
  resource: Pick<InstanceResource, 'source'>,
): boolean {
  return resource.source === 'upload';
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

function instanceResourceEffectiveSide(
  resource: Pick<InstanceResource, 'side_override'>,
  catalogSide?: ModSyncSide,
): ModSyncSide {
  const override = resource.side_override?.trim();
  if (override === 'client' || override === 'server' || override === 'both') {
    return override;
  }
  return catalogSide ?? 'unknown';
}

function contentTargetForSide(side: ModSyncSide, resourceType: string): ContentTarget {
  switch (resourceType) {
    case 'resourcepack':
      return side === 'client' ? 'client-resourcepacks' : 'resourcepacks';
    case 'shader':
      return side === 'client' ? 'client-shaders' : 'shaderpacks';
    default:
      return side === 'client' ? 'client-mods' : 'mods';
  }
}

export function instanceResourceContentTarget(
  resource: Pick<InstanceResource, 'side_override' | 'resource_type'>,
  catalogSide?: ModSyncSide,
): ContentTarget {
  return contentTargetForSide(
    instanceResourceEffectiveSide(resource, catalogSide),
    resource.resource_type ?? 'mod',
  );
}

export function instanceResourceModTarget(
  resource: Pick<InstanceResource, 'side_override' | 'resource_type'>,
  catalogSide?: ModSyncSide,
): ModTarget {
  const target = instanceResourceContentTarget(resource, catalogSide);
  if (target === 'client-mods' || target === 'mods') {
    return target;
  }
  return 'mods';
}

export function gameServerInstallSide(side?: ModSyncSide): ModSyncSide {
  if (side === 'client' || side === 'server' || side === 'both') {
    return side;
  }
  return 'both';
}

export function isServerOnlyMod(item: Pick<ModCatalogItem, 'client_side' | 'server_side'>): boolean {
  return modSyncSide(item) === 'server';
}

export function needsServerRestartAfterSync(target?: string): boolean {
  switch (target) {
    case 'client-mods':
    case 'client-resourcepacks':
    case 'client-shaders':
      return false;
    default:
      return true;
  }
}

export function mergeServerModLists(
  serverMods: GameServerFileEntry[],
  clientMods: GameServerFileEntry[],
): GameServerFileEntry[] {
  return [...serverMods, ...clientMods];
}

export function instanceResourceSupportsServerSync(resource: InstanceResource): boolean {
  if (!['mod', 'resourcepack', 'shader'].includes(resource.resource_type)) return false;
  if (resource.source === 'upload') {
    return Boolean(resource.filename);
  }
  return Boolean(resource.project_id && resource.version_id);
}

export function applyModTargetToBodies(
  bodies: GameServerContentSyncBody[],
  modTarget: ContentTarget,
): GameServerContentSyncBody[] {
  if (modTarget === 'mods' || modTarget === 'resourcepacks' || modTarget === 'shaderpacks') return bodies;
  return bodies.map((body) => ({ ...body, mod_target: modTarget }));
}
