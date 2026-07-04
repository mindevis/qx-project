import {
  type ModDependency,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { cachedGetModVersion, cachedListModVersions } from '@/lib/modCatalogCache';

export type ModDirectDependencies = {
  installVersion: ModVersion;
  required: ModDependency[];
  optional: ModDependency[];
};

export type EnrichModDepsParams = {
  loader?: string;
  mcVersion: string;
};

export function dependencyProjectKey(dep: ModDependency): string | undefined {
  if (!dep.project_id) return undefined;
  return `${dep.source}:${dep.project_id}`;
}

export function isDependencyResolved(dep: ModDependency): boolean {
  return Boolean(dep.project_id && dep.version_id && dep.filename && dep.download_url);
}

export async function enrichModDependency(
  dep: ModDependency,
  params: EnrichModDepsParams,
): Promise<ModDependency> {
  if (isDependencyResolved(dep)) {
    return dep;
  }
  if (!dep.project_id) {
    return dep;
  }
  const source = dep.source as ModSource;
  const query = { loader: params.loader, mc_version: params.mcVersion };
  try {
    let versionId = dep.version_id;
    if (!versionId) {
      const versions = await cachedListModVersions(source, dep.project_id, query);
      versionId = versions[0]?.id;
    }
    if (!versionId) {
      return dep;
    }
    const detail = await cachedGetModVersion(source, dep.project_id, versionId, query);
    const file = detail.files[0];
    if (!file?.url) {
      return dep;
    }
    return {
      ...dep,
      version_id: detail.id,
      version_number: detail.version_number ?? dep.version_number,
      filename: file.filename,
      download_url: file.url,
      file_size: file.size ?? dep.file_size,
    };
  } catch {
    return dep;
  }
}

export async function enrichModDependencies(
  dependencies: ModDependency[],
  params: EnrichModDepsParams,
): Promise<ModDependency[]> {
  return Promise.all(dependencies.map((dep) => enrichModDependency(dep, params)));
}

export async function loadModDirectDependencies(
  source: ModSource,
  projectId: string,
  version: ModVersion,
  params: EnrichModDepsParams,
): Promise<ModDirectDependencies> {
  const detail = await cachedGetModVersion(source, projectId, version.id, {
    loader: params.loader,
    mc_version: params.mcVersion,
  });
  if (!detail.files[0]?.url) {
    return { installVersion: detail, required: [], optional: [] };
  }
  const directRequired = (detail.dependencies ?? []).filter((d) => d.dependency_type === 'required');
  const directOptional = (detail.dependencies ?? []).filter((d) => d.dependency_type === 'optional');
  const [required, optional] = await Promise.all([
    enrichModDependencies(directRequired, params),
    enrichModDependencies(directOptional, params),
  ]);
  return { installVersion: detail, required, optional };
}

export type ModDepsWizardStep = {
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  resourceType: ModProjectType;
};

function dependencyToWizardStep(dep: ModDependency): ModDepsWizardStep | null {
  if (!dep.project_id || !isDependencyResolved(dep)) return null;
  return {
    source: dep.source as ModSource,
    projectId: dep.project_id,
    projectName: dep.project_name ?? dep.project_id,
    version: {
      id: dep.version_id!,
      version_number: dep.version_number ?? dep.version_id!,
      files: [{ filename: dep.filename!, url: dep.download_url!, size: dep.file_size }],
    },
    resourceType: 'mod',
  };
}

function dependencyHasPendingDeps(
  dependencies: ModDependency[],
  installedProjectIds: Set<string>,
): boolean {
  return dependencies.some((dep) => {
    if (!dep.project_id) return false;
    const key = dependencyProjectKey(dep);
    return !(key && installedProjectIds.has(key));
  });
}

/** Required deps selected in a step that have their own dependency tree needing review. */
export async function buildNestedDepsWizardSteps(
  selectedRequired: ModDependency[],
  installedProjectIds: Set<string>,
  params: EnrichModDepsParams,
): Promise<ModDepsWizardStep[]> {
  const steps: ModDepsWizardStep[] = [];
  const seen = new Set<string>();

  for (const dep of selectedRequired) {
    if (dep.dependency_type !== 'required' || !dep.project_id) continue;
    const key = dependencyProjectKey(dep);
    if (key && installedProjectIds.has(key)) continue;
    const enriched = await enrichModDependency(dep, params);
    if (!isDependencyResolved(enriched)) continue;
    if (key && seen.has(key)) continue;

    try {
      const detail = await cachedGetModVersion(
        enriched.source as ModSource,
        enriched.project_id,
        enriched.version_id!,
        { loader: params.loader, mc_version: params.mcVersion },
      );
      const nested = detail.dependencies ?? [];
      if (!dependencyHasPendingDeps(nested, installedProjectIds)) continue;
      const step = dependencyToWizardStep(enriched);
      if (!step) continue;
      seen.add(key!);
      steps.push(step);
    } catch {
      // Skip nested step when version metadata cannot be loaded.
    }
  }
  return steps;
}

export function dedupeInstallItemsByProject<T extends { source: ModSource; projectId: string }>(
  items: T[],
): T[] {
  const seen = new Set<string>();
  const result: T[] = [];
  for (const item of items) {
    const key = `${item.source}:${item.projectId}`;
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(item);
  }
  return result;
}

/** Walks required dependencies recursively; result is install order (leaves first). */
export async function collectTransitiveRequiredDependencies(
  rootDependencies: ModDependency[],
  params: EnrichModDepsParams,
): Promise<ModDependency[]> {
  const seen = new Set<string>();
  const ordered: ModDependency[] = [];

  async function visit(dep: ModDependency): Promise<void> {
    if (dep.dependency_type !== 'required' || !dep.project_id) {
      return;
    }
    const enriched = await enrichModDependency(dep, params);
    const key = dependencyProjectKey(enriched);
    if (!key || seen.has(key)) {
      return;
    }
    seen.add(key);

    if (isDependencyResolved(enriched)) {
      try {
        const detail = await cachedGetModVersion(
          enriched.source as ModSource,
          enriched.project_id,
          enriched.version_id!,
          { loader: params.loader, mc_version: params.mcVersion },
        );
        const nested = (detail.dependencies ?? []).filter((d) => d.dependency_type === 'required');
        for (const child of nested) {
          await visit(child);
        }
      } catch {
        // Nested lookup failed; keep this dependency entry for UI/errors.
      }
    }
    ordered.push(enriched);
  }

  for (const dep of rootDependencies) {
    await visit(dep);
  }
  return ordered;
}

export function unresolvedSelectedDependencies(
  dependencies: ModDependency[],
  installedProjectIds: Set<string>,
  requiredSelected: Set<string>,
  optionalSelected: Set<string>,
): ModDependency[] {
  return dependencies.filter((dep) => {
    if (!dep.project_id) return false;
    const key = dependencyProjectKey(dep);
    if (key && installedProjectIds.has(key)) return false;
    const selected =
      dep.dependency_type === 'required'
        ? requiredSelected.has(dep.project_id)
        : optionalSelected.has(dep.project_id);
    return selected && !isDependencyResolved(dep);
  });
}
