import { describe, expect, it } from 'vitest';
import { filterVersionsForCatalogLoader, selectLatestCompatibleVersion } from './selectLatestModVersion';

const older = {
  id: 'old',
  version_number: '1.0.0',
  game_versions: ['1.21.1'],
  loaders: ['neoforge'],
  files: [],
  published_at: '2025-01-01T00:00:00Z',
};

const newest = {
  id: 'new',
  version_number: '2.0.0',
  game_versions: ['1.21.1'],
  loaders: ['neoforge'],
  files: [],
  published_at: '2026-08-01T00:00:00Z',
};

const fabricOnly = {
  id: 'fabric',
  version_number: '3.0.0',
  game_versions: ['1.21.1'],
  loaders: ['fabric'],
  files: [],
  published_at: '2026-08-10T00:00:00Z',
};

const datapack = {
  id: 'datapack',
  version_number: '5.3.0',
  game_versions: ['1.21.1'],
  loaders: ['datapack'],
  files: [{ filename: 'Dungeons and Taverns v5.3.0.zip', url: 'https://cdn/pack.zip' }],
  published_at: '2026-08-12T00:00:00Z',
};

describe('selectLatestCompatibleVersion', () => {
  it('picks the newest matching loader and Minecraft version', () => {
    const latest = selectLatestCompatibleVersion(
      [older, fabricOnly, newest],
      'neoforge',
      '1.21.1',
    );
    expect(latest?.id).toBe('new');
  });

  it('does not fall back to another loader when nothing matches', () => {
    const latest = selectLatestCompatibleVersion([older, newest], 'quilt', '1.20.1');
    expect(latest).toBeUndefined();
  });

  it('keeps datapack versions and drops fabric/forge jars', () => {
    const latest = selectLatestCompatibleVersion(
      [fabricOnly, datapack, newest],
      'datapack',
      '1.21.1',
    );
    expect(latest?.id).toBe('datapack');
  });

  it('keeps CurseForge files that omit loader metadata', () => {
    const curseforge = {
      id: 'cf',
      version_number: '19.8.2',
      game_versions: ['1.21.1', 'Forge', 'Client'],
      loaders: [''],
      files: [{ filename: 'jei-1.21.1-forge.jar', url: 'https://cdn/jei.jar' }],
      published_at: '2026-08-01T00:00:00Z',
    };
    expect(filterVersionsForCatalogLoader([curseforge], 'forge')).toEqual([curseforge]);
    expect(selectLatestCompatibleVersion([curseforge], 'forge', '1.21.1')?.id).toBe('cf');
  });
});
