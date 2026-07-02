import { describe, expect, it } from 'vitest';
import {
  CONFIG_EXTENSIONS,
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
        'config/fabric-api.toml',
        'config/sodium-options.json',
        'config/unknown-mod.cfg',
      ],
    );

    expect(result.groups).toHaveLength(2);
    expect(result.groups[0].paths).toEqual(['config/fabric-api.toml']);
    expect(result.groups[1].paths).toEqual(['config/sodium-options.json']);
    expect(result.other).toEqual(['config/unknown-mod.cfg']);
  });

  it('matches nested config paths one level deep', () => {
    const result = groupConfigFilesByMod([fabricMod], ['config/fabric-api/client.json']);
    expect(result.groups[0]?.paths).toEqual(['config/fabric-api/client.json']);
    expect(result.other).toEqual([]);
  });
});

describe('listConfigPaths', () => {
  it('lists top-level and one-level nested config files', async () => {
    const paths = await listConfigPaths(async (path) => {
      if (path === 'config') {
        return [
          { path: 'config/server.properties', dir: false, name: 'server.properties' },
          { path: 'config/fabric-api', dir: true, name: 'fabric-api' },
        ];
      }
      if (path === 'config/fabric-api') {
        return [{ path: 'config/fabric-api/client.json', dir: false, name: 'client.json' }];
      }
      return [];
    });

    expect(paths).toEqual(['config/fabric-api/client.json', 'config/server.properties']);
  });
});
