import { describe, expect, it, vi } from 'vitest';
import { api } from '@/api/client';
import {
  buildNestedDepsWizardSteps,
  collectTransitiveRequiredDependencies,
  dedupeInstallItemsByProject,
  enrichModDependency,
  isDependencyResolved,
  loadModDirectDependencies,
  unresolvedSelectedDependencies,
} from '@/lib/modCatalogDeps';

describe('modCatalogDeps', () => {
  it('detects resolved dependency', () => {
    expect(
      isDependencyResolved({
        source: 'curseforge',
        project_id: 'dep-1',
        version_id: 'v1',
        filename: 'dep.jar',
        download_url: 'https://example/dep.jar',
      }),
    ).toBe(true);
  });

  it('enriches missing download url from mod version api', async () => {
    vi.spyOn(api, 'listModVersions').mockResolvedValue({
      items: [{ id: 'v1', version_number: '1.0.0', files: [] }],
    });
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'v1',
      version_number: '1.0.0',
      files: [{ filename: 'nirvana.jar', url: 'https://example/nirvana.jar', size: 100 }],
    });

    const enriched = await enrichModDependency(
      {
        source: 'curseforge',
        project_id: 'nirvana',
        dependency_type: 'required',
      },
      { loader: 'forge', mcVersion: '1.21.1' },
    );

    expect(enriched.download_url).toBe('https://example/nirvana.jar');
    expect(enriched.filename).toBe('nirvana.jar');
  });

  it('lists unresolved selected dependencies', () => {
    const unresolved = unresolvedSelectedDependencies(
      [
        {
          source: 'curseforge',
          project_id: 'dep-1',
          dependency_type: 'required',
        },
      ],
      new Set(),
      new Set(['dep-1']),
      new Set(),
    );
    expect(unresolved).toHaveLength(1);
  });

  it('collects nested required dependencies in install order', async () => {
    vi.spyOn(api, 'listModVersions').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'getModVersion').mockImplementation(async (_source, projectId) => {
      if (projectId === 'main-dep') {
        return {
          id: 'v-main',
          version_number: '1.0.0',
          files: [{ filename: 'main.jar', url: 'https://example/main.jar', size: 1 }],
          dependencies: [
            {
              source: 'curseforge',
              project_id: 'leaf-dep',
              dependency_type: 'required',
              version_id: 'v-leaf',
              filename: 'leaf.jar',
              download_url: 'https://example/leaf.jar',
            },
          ],
        };
      }
      return {
        id: 'v-leaf',
        version_number: '1.0.0',
        files: [{ filename: 'leaf.jar', url: 'https://example/leaf.jar', size: 1 }],
        dependencies: [],
      };
    });

    const ordered = await collectTransitiveRequiredDependencies(
      [
        {
          source: 'curseforge',
          project_id: 'main-dep',
          dependency_type: 'required',
          version_id: 'v-main',
          filename: 'main.jar',
          download_url: 'https://example/main.jar',
        },
      ],
      { loader: 'forge', mcVersion: '1.21.1' },
    );

    expect(ordered.map((dep) => dep.project_id)).toEqual(['leaf-dep', 'main-dep']);
  });

  it('loads only direct dependencies for a mod version', async () => {
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'v-root',
      version_number: '1.0.0',
      files: [{ filename: 'root.jar', url: 'https://example/root.jar', size: 1 }],
      dependencies: [
        {
          source: 'curseforge',
          project_id: 'direct-req',
          dependency_type: 'required',
          version_id: 'v-req',
          filename: 'req.jar',
          download_url: 'https://example/req.jar',
        },
        {
          source: 'curseforge',
          project_id: 'direct-opt',
          dependency_type: 'optional',
          version_id: 'v-opt',
          filename: 'opt.jar',
          download_url: 'https://example/opt.jar',
        },
        {
          source: 'modrinth',
          project_id: 'tq47Uqpn',
          dependency_type: 'incompatible',
        },
      ],
    });

    const loaded = await loadModDirectDependencies(
      'curseforge',
      'root',
      { id: 'v-root', version_number: '1.0.0', files: [] },
      { loader: 'forge', mcVersion: '1.21.1' },
    );

    expect(loaded.required.map((dep) => dep.project_id)).toEqual(['direct-req']);
    expect(loaded.optional.map((dep) => dep.project_id)).toEqual(['direct-opt']);
    expect(loaded.required.some((dep) => dep.project_id === 'tq47Uqpn')).toBe(false);
  });

  it('queues nested wizard steps for required deps with pending sub-dependencies', async () => {
    vi.spyOn(api, 'getModVersion').mockImplementation(async (_source, projectId) => {
      if (projectId === 'parent-dep') {
        return {
          id: 'v-parent',
          version_number: '1.0.0',
          files: [{ filename: 'parent.jar', url: 'https://example/parent.jar', size: 1 }],
          dependencies: [
            {
              source: 'curseforge',
              project_id: 'leaf-dep',
              dependency_type: 'required',
              version_id: 'v-leaf',
              filename: 'leaf.jar',
              download_url: 'https://example/leaf.jar',
            },
            {
              source: 'curseforge',
              project_id: 'optional-dep',
              dependency_type: 'optional',
              version_id: 'v-opt',
              filename: 'opt.jar',
              download_url: 'https://example/opt.jar',
            },
          ],
        };
      }
      return {
        id: 'v-leaf',
        version_number: '1.0.0',
        files: [{ filename: 'leaf.jar', url: 'https://example/leaf.jar', size: 1 }],
        dependencies: [],
      };
    });

    const steps = await buildNestedDepsWizardSteps(
      [
        {
          source: 'curseforge',
          project_id: 'parent-dep',
          dependency_type: 'required',
          version_id: 'v-parent',
          filename: 'parent.jar',
          download_url: 'https://example/parent.jar',
        },
      ],
      new Set(),
      { loader: 'forge', mcVersion: '1.21.1' },
    );

    expect(steps.map((step) => step.projectId)).toEqual(['parent-dep']);
  });

  it('dedupes install items by project', () => {
    const items = dedupeInstallItemsByProject([
      { source: 'curseforge', projectId: 'a' },
      { source: 'curseforge', projectId: 'b' },
      { source: 'curseforge', projectId: 'a' },
    ]);
    expect(items.map((item) => item.projectId)).toEqual(['a', 'b']);
  });
});
