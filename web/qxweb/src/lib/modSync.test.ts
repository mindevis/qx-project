import { describe, expect, it } from 'vitest';
import {
  buildModSyncBodies,
  applyModTargetToBodies,
  instanceResourceModTarget,
  instanceResourceContentTarget,
  contentTargetFromPath,
  gameServerInstallSide,
  instanceResourceSupportsServerSync,
  isServerOnlyMod,
  needsServerRestartAfterSync,
  instanceResourceVersionKey,
  isInstanceResourceOnServer,
  isModOnServer,
  isFilenameOnServer,
  isCatalogItemOnServer,
  isCatalogItemInstalledOnInstance,
  modSupportsServerSync,
  modSyncSide,
} from './modSync';

describe('buildModSyncBodies', () => {
  it('includes installed required dependencies before the main mod', () => {
    const bodies = buildModSyncBodies(
      {
        source: 'modrinth',
        projectId: 'main-mod',
        projectName: 'Main Mod',
        version: {
          id: 'main-ver',
          version_number: '1.0.0',
          files: [{ filename: 'main.jar', url: 'https://example/main.jar' }],
        },
      },
      [
        {
          source: 'modrinth',
          project_id: 'dep-mod',
          project_name: 'Dep Mod',
          version_id: 'dep-ver',
          filename: 'dep.jar',
          resource_type: 'mod',
          installed_at: 'now',
        },
      ],
      {
        id: 'main-ver',
        version_number: '1.0.0',
        files: [{ filename: 'main.jar', url: 'https://example/main.jar' }],
        dependencies: [
          {
            source: 'modrinth',
            project_id: 'dep-mod',
            project_name: 'Dep Mod',
            dependency_type: 'required',
            version_id: 'dep-ver',
            filename: 'dep.jar',
            download_url: 'https://example/dep.jar',
          },
        ],
      },
    );

    expect(bodies).toHaveLength(2);
    expect(bodies[0].project_id).toBe('dep-mod');
    expect(bodies[1].project_id).toBe('main-mod');
  });

  it('skips required dependencies that are not installed on the instance', () => {
    const bodies = buildModSyncBodies(
      {
        source: 'modrinth',
        projectId: 'main-mod',
        projectName: 'Main Mod',
        version: {
          id: 'main-ver',
          version_number: '1.0.0',
          files: [{ filename: 'main.jar', url: 'https://example/main.jar' }],
        },
      },
      [],
      {
        id: 'main-ver',
        version_number: '1.0.0',
        files: [{ filename: 'main.jar', url: 'https://example/main.jar' }],
        dependencies: [
          {
            source: 'modrinth',
            project_id: 'dep-mod',
            dependency_type: 'required',
            version_id: 'dep-ver',
            filename: 'dep.jar',
            download_url: 'https://example/dep.jar',
          },
        ],
      },
    );

    expect(bodies).toHaveLength(1);
    expect(bodies[0].project_id).toBe('main-mod');
  });
});

describe('modSyncSide', () => {
  it('detects server-side mods', () => {
    expect(modSyncSide({ client_side: 'unsupported', server_side: 'required' })).toBe('server');
    expect(modSupportsServerSync({ client_side: 'unsupported', server_side: 'required' })).toBe(true);
  });

  it('detects client-only mods', () => {
    expect(modSyncSide({ client_side: 'required', server_side: 'unsupported' })).toBe('client');
    expect(modSupportsServerSync({ client_side: 'required', server_side: 'unsupported' })).toBe(false);
  });

  it('treats blank side metadata as unknown', () => {
    expect(modSyncSide({ client_side: '', server_side: '' })).toBe('unknown');
  });
});

describe('needsServerRestartAfterSync', () => {
  it('skips restart for client-only sync targets', () => {
    expect(needsServerRestartAfterSync('client-mods')).toBe(false);
    expect(needsServerRestartAfterSync('client-resourcepacks')).toBe(false);
    expect(needsServerRestartAfterSync('client-shaders')).toBe(false);
  });

  it('skips restart for resource packs and shaders the server process does not load', () => {
    expect(needsServerRestartAfterSync('resourcepacks')).toBe(false);
    expect(needsServerRestartAfterSync('shaderpacks')).toBe(false);
  });

  it('requires restart for server-loaded content', () => {
    expect(needsServerRestartAfterSync('mods')).toBe(true);
    expect(needsServerRestartAfterSync(undefined)).toBe(true);
  });
});

describe('isModOnServer', () => {
  it('matches filename on server', () => {
    const onServer = isModOnServer(
      [{ name: 'sodium-0.5.0.jar', path: 'mods/sodium-0.5.0.jar', dir: false }],
      { files: [{ filename: 'sodium-0.5.0.jar', url: 'https://example/mod.jar' }] },
    );
    expect(onServer).toBe(true);
  });

  it('is case-insensitive', () => {
    const onServer = isModOnServer(
      [{ name: 'Mod.JAR', path: 'mods/Mod.JAR', dir: false }],
      { files: [{ filename: 'mod.jar', url: 'https://example/mod.jar' }] },
    );
    expect(onServer).toBe(true);
  });

  it('treats a .disabled jar as still installed', () => {
    expect(
      isModOnServer(
        [{ name: 'sodium-0.5.0.jar.disabled', path: 'mods/sodium-0.5.0.jar.disabled', dir: false }],
        { files: [{ filename: 'sodium-0.5.0.jar', url: 'https://example/mod.jar' }] },
      ),
    ).toBe(true);
    expect(
      isFilenameOnServer(
        [{ name: 'sodium-0.5.0.jar.disabled', path: 'mods/sodium-0.5.0.jar.disabled', dir: false }],
        'sodium-0.5.0.jar',
      ),
    ).toBe(true);
  });
});

describe('instanceResourceVersionKey', () => {
  it('builds a stable cache key', () => {
    expect(
      instanceResourceVersionKey({
        source: 'curseforge',
        project_id: 'journeymap',
        version_id: 'ver-1',
      }),
    ).toBe('curseforge:journeymap:ver-1');
  });

  it('builds a key for uploaded mods by filename', () => {
    expect(
      instanceResourceVersionKey({
        source: 'upload',
        filename: 'custom-mod.jar',
      }),
    ).toBe('upload:custom-mod.jar');
  });
});

describe('instanceResourceModTarget', () => {
  it('routes client-only resourcepack override to client-resourcepacks folder', () => {
    expect(
      instanceResourceContentTarget({ side_override: 'client', resource_type: 'resourcepack' }),
    ).toBe('client-resourcepacks');
  });

  it('routes client-only shader override to client-shaders folder', () => {
    expect(instanceResourceContentTarget({ side_override: 'client', resource_type: 'shader' })).toBe(
      'client-shaders',
    );
  });

  it('routes client-only override to client-mods folder', () => {
    expect(instanceResourceModTarget({ side_override: 'client' })).toBe('client-mods');
    expect(instanceResourceModTarget({ side_override: 'server' })).toBe('mods');
    expect(instanceResourceModTarget({ side_override: 'both' })).toBe('mods');
  });
});

describe('contentTargetFromPath', () => {
  it('maps installed folders to content targets', () => {
    expect(contentTargetFromPath('client-resourcepacks/pack.zip')).toBe('client-resourcepacks');
    expect(contentTargetFromPath('resourcepacks/pack.zip')).toBe('resourcepacks');
    expect(contentTargetFromPath('shaderpacks/shaders.zip')).toBe('shaderpacks');
    expect(contentTargetFromPath('client-shaders/shaders.zip')).toBe('client-shaders');
  });
});

describe('gameServerInstallSide', () => {
  it('keeps explicit sides and defaults unknown catalog data to both', () => {
    expect(gameServerInstallSide('client')).toBe('client');
    expect(gameServerInstallSide('server')).toBe('server');
    expect(gameServerInstallSide('unknown')).toBe('both');
    expect(gameServerInstallSide()).toBe('both');
  });
});

describe('isServerOnlyMod', () => {
  it('detects server-only catalog entries', () => {
    expect(isServerOnlyMod({ client_side: 'unsupported', server_side: 'required' })).toBe(true);
  });
});

describe('applyModTargetToBodies', () => {
  it('adds mod_target to sync bodies', () => {
    const bodies = applyModTargetToBodies(
      [{ source: 'modrinth', project_id: 'a', version_id: 'v', filename: 'a.jar', download_url: 'https://x' }],
      'client-mods',
    );
    expect(bodies[0].mod_target).toBe('client-mods');
  });
});

describe('instanceResourceSupportsServerSync', () => {
  it('allows catalog mods with project and version ids', () => {
    expect(
      instanceResourceSupportsServerSync({
        source: 'modrinth',
        project_id: 'sodium',
        version_id: 'ver-1',
        project_name: 'Sodium',
        filename: 'sodium.jar',
        resource_type: 'mod',
        installed_at: 'now',
      }),
    ).toBe(true);
  });

  it('allows uploaded mods with filename only', () => {
    expect(
      instanceResourceSupportsServerSync({
        source: 'upload',
        project_name: 'Custom Mod',
        filename: 'custom-mod.jar',
        resource_type: 'mod',
        installed_at: 'now',
      }),
    ).toBe(true);
  });

  it('allows uploaded resource packs and shaders with filename', () => {
    expect(
      instanceResourceSupportsServerSync({
        source: 'upload',
        project_name: 'Pack',
        filename: 'pack.zip',
        resource_type: 'resourcepack',
        installed_at: 'now',
      }),
    ).toBe(true);
    expect(
      instanceResourceSupportsServerSync({
        source: 'upload',
        project_name: 'Shader',
        filename: 'shader.zip',
        resource_type: 'shader',
        installed_at: 'now',
      }),
    ).toBe(true);
  });

  it('rejects unsupported resource types', () => {
    expect(
      instanceResourceSupportsServerSync({
        source: 'upload',
        project_name: 'Pack',
        filename: 'pack.zip',
        resource_type: 'datapack',
        installed_at: 'now',
      }),
    ).toBe(false);
  });
});

describe('isInstanceResourceOnServer', () => {
  const serverFiles = [{ name: 'actual-mod-file.jar', path: 'mods/actual-mod-file.jar', dir: false }];

  it('matches instance filename by default', () => {
    expect(
      isInstanceResourceOnServer(serverFiles, { filename: 'actual-mod-file.jar' }),
    ).toBe(true);
  });

  it('matches any cached version filename instead of instance filename', () => {
    expect(
      isInstanceResourceOnServer(
        serverFiles,
        { filename: 'launcher-local-name.jar' },
        ['actual-mod-file.jar'],
      ),
    ).toBe(true);
    expect(
      isInstanceResourceOnServer(
        serverFiles,
        { filename: 'launcher-local-name.jar' },
        ['other-file.jar'],
      ),
    ).toBe(false);
  });
});

describe('isCatalogItemOnServer', () => {
  it('matches catalog slug or name against installed filenames', () => {
    const installed = [{ name: 'sodium-fabric-0.5.8.jar', path: 'mods/sodium-fabric-0.5.8.jar', dir: false }];
    expect(
      isCatalogItemOnServer({ id: 'AANobbMI', slug: 'sodium', name: 'Sodium' }, installed),
    ).toBe(true);
    expect(
      isCatalogItemOnServer({ id: 'jei', slug: 'jei', name: 'JEI' }, installed),
    ).toBe(false);
  });

  it('still matches a disabled jar on the server', () => {
    const installed = [
      { name: 'sodium-fabric-0.5.8.jar.disabled', path: 'mods/sodium-fabric-0.5.8.jar.disabled', dir: false },
    ];
    expect(
      isCatalogItemOnServer({ id: 'AANobbMI', slug: 'sodium', name: 'Sodium' }, installed),
    ).toBe(true);
  });
});

describe('isCatalogItemInstalledOnInstance', () => {
  it('matches a catalog project id on the instance', () => {
    expect(
      isCatalogItemInstalledOnInstance(
        { id: 'sodium', slug: 'sodium', name: 'Sodium', source: 'modrinth' },
        [{ source: 'modrinth', project_id: 'sodium', filename: 'sodium.jar' }],
      ),
    ).toBe(true);
  });

  it('matches a connect-copied upload by filename even without project id', () => {
    expect(
      isCatalogItemInstalledOnInstance(
        { id: '250498', slug: 'mowzies-mobs', name: "Mowzie's Mobs", source: 'curseforge' },
        [{ source: 'upload', filename: "Mowzie's Mobs-1.20.1-1.7.3.jar" }],
      ),
    ).toBe(true);
  });

  it('does not match an unrelated upload', () => {
    expect(
      isCatalogItemInstalledOnInstance(
        { id: 'jei', slug: 'jei', name: 'JEI', source: 'curseforge' },
        [{ source: 'upload', filename: 'sodium-fabric-0.5.8.jar' }],
      ),
    ).toBe(false);
  });
});
