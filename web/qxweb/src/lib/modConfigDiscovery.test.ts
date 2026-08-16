import { describe, expect, it } from 'vitest';
import {
  CONFIG_EXTENSIONS,
  configFileExtension,
  configRelativePath,
  filterConfigFileEntries,
  filterGroupedConfigs,
  groupConfigFilesByMod,
  isConfigFilePath,
  listConfigPaths,
  type ModConfigMod,
} from './modConfigDiscovery';

describe('isConfigFilePath', () => {
  it('recognizes supported config extensions', () => {
    for (const ext of CONFIG_EXTENSIONS) {
      expect(isConfigFilePath(`config/example${ext}`)).toBe(true);
    }
    expect(isConfigFilePath('config/readme.txt')).toBe(false);
  });
});

describe('configRelativePath', () => {
  it('strips the config prefix', () => {
    expect(configRelativePath('config/fabric-api/client.json')).toBe('fabric-api/client.json');
  });
});

describe('filterConfigFileEntries', () => {
  it('filters by file name and relative path', () => {
    const entries = [
      { path: 'config/sodium-options.json' },
      { path: 'config/fabric-api/client.json' },
    ];
    expect(filterConfigFileEntries(entries, 'sodium')).toEqual([{ path: 'config/sodium-options.json' }]);
    expect(filterConfigFileEntries(entries, 'fabric-api')).toEqual([
      { path: 'config/fabric-api/client.json' },
    ]);
    expect(filterConfigFileEntries(entries, '')).toEqual(entries);
  });
});

describe('groupConfigFilesByMod', () => {
  const fabricMod: ModConfigMod = {
    key: 'fabric-api',
    label: 'Fabric API',
    filename: 'fabric-api-0.100.0+1.21.jar',
    project_name: 'Fabric API',
  };

  const sodiumMod: ModConfigMod = {
    key: 'sodium',
    label: 'Sodium',
    filename: 'sodium-fabric-0.5.0.jar',
    project_name: 'Sodium',
  };

  it('groups config files by jar base name and project name', () => {
    const result = groupConfigFilesByMod(
      [fabricMod, sodiumMod],
      [
        { path: 'config/fabric-api.toml', size: 128 },
        { path: 'config/sodium-options.json', size: 256 },
        { path: 'config/unknown-mod.cfg' },
      ],
    );

    expect(result.groups).toHaveLength(2);
    expect(result.groups[0].files).toEqual([{ path: 'config/fabric-api.toml', size: 128 }]);
    expect(result.groups[1].files).toEqual([{ path: 'config/sodium-options.json', size: 256 }]);
    expect(result.other).toEqual([{ path: 'config/unknown-mod.cfg' }]);
  });

  it('matches nested config paths one level deep', () => {
    const result = groupConfigFilesByMod([fabricMod], [{ path: 'config/fabric-api/client.json' }]);
    expect(result.groups[0]?.files).toEqual([{ path: 'config/fabric-api/client.json' }]);
    expect(result.other).toEqual([]);
  });
});

describe('filterGroupedConfigs', () => {
  const fabricMod: ModConfigMod = {
    key: 'fabric-api',
    label: 'Fabric API',
    filename: 'fabric-api.jar',
    project_name: 'Fabric API',
  };

  it('keeps a whole group when the mod name matches', () => {
    const grouped = groupConfigFilesByMod(
      [fabricMod],
      [
        { path: 'config/fabric-api.toml' },
        { path: 'config/unknown.cfg' },
      ],
    );
    const filtered = filterGroupedConfigs(grouped, 'fabric', 'Other');
    expect(filtered.groups).toHaveLength(1);
    expect(filtered.groups[0].files).toHaveLength(1);
    expect(filtered.other).toEqual([]);
  });

  it('filters other files by path and ignores unmatched mods', () => {
    const grouped = groupConfigFilesByMod(
      [fabricMod],
      [
        { path: 'config/fabric-api.toml' },
        { path: 'config/sodium-options.json' },
      ],
    );
    const filtered = filterGroupedConfigs(grouped, 'sodium', 'Other');
    expect(filtered.groups).toEqual([]);
    expect(filtered.other).toEqual([{ path: 'config/sodium-options.json' }]);
  });
});

describe('configFileExtension', () => {
  it('returns the lowercase extension', () => {
    expect(configFileExtension('config/sodium-options.JSON')).toBe('json');
    expect(configFileExtension('config/readme')).toBe('');
  });
});

describe('listConfigPaths', () => {
  it('lists top-level and nested config files with sizes', async () => {
    const paths = await listConfigPaths(async (path) => {
      if (path === 'config') {
        return [
          { path: 'config/server.properties', dir: false, name: 'server.properties', size: 64 },
          { path: 'config/fabric-api', dir: true, name: 'fabric-api' },
        ];
      }
      if (path === 'config/fabric-api') {
        return [
          { path: 'config/fabric-api/client.json', dir: false, name: 'client.json', size: 32 },
          { path: 'config/fabric-api/nested', dir: true, name: 'nested' },
        ];
      }
      if (path === 'config/fabric-api/nested') {
        return [{ path: 'config/fabric-api/nested/extra.toml', dir: false, name: 'extra.toml', size: 16 }];
      }
      return [];
    });

    expect(paths).toEqual([
      { path: 'config/fabric-api/client.json', size: 32 },
      { path: 'config/fabric-api/nested/extra.toml', size: 16 },
      { path: 'config/server.properties', size: 64 },
    ]);
  });
});
