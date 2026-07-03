import { type InstanceResource } from '@/api/client';
import { cachedGetModProject } from '@/lib/modCatalogCache';

export function instanceResourceIconKey(
  item: Pick<InstanceResource, 'source' | 'project_id'>,
): string | undefined {
  if (!item.project_id) return undefined;
  return `${item.source}:${item.project_id}`;
}

export async function fetchMissingResourceIcons(
  items: InstanceResource[],
): Promise<Record<string, string>> {
  const missing = items.filter((item) => item.project_id && !item.icon_url?.trim());
  if (missing.length === 0) return {};

  const seen = new Set<string>();
  const results: Record<string, string> = {};

  await Promise.all(
    missing.map(async (item) => {
      const key = instanceResourceIconKey(item);
      if (!key || seen.has(key)) return;
      seen.add(key);
      try {
        const project = await cachedGetModProject(item.source, item.project_id!);
        if (project.icon_url?.trim()) {
          results[key] = project.icon_url;
        }
      } catch {
        // Catalog metadata is optional for display.
      }
    }),
  );

  return results;
}

export async function fetchModProjectIcons(
  projects: Array<{ source: import('@/api/client').ModSource; projectId: string }>,
): Promise<Map<string, string>> {
  const results = new Map<string, string>();
  const seen = new Set<string>();

  await Promise.all(
    projects.map(async ({ source, projectId }) => {
      const key = `${source}:${projectId}`;
      if (seen.has(key)) return;
      seen.add(key);
      try {
        const project = await cachedGetModProject(source, projectId);
        if (project.icon_url?.trim()) {
          results.set(key, project.icon_url);
        }
      } catch {
        // Ignore lookup failures during install enrichment.
      }
    }),
  );

  return results;
}
