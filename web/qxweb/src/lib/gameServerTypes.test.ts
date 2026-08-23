import { describe, expect, it } from 'vitest';
import {
  ALL_GAME_SERVER_TYPES,
  DEFAULT_GAME_SERVER_TYPE,
  gameServerCatalogTabs,
  gameServerSupportsMods,
  gameServerSupportsPlugins,
  gameServerTypeCapabilities,
  gameServerTypeLabelText,
  isKnownGameServerType,
  pluginCatalogSources,
  catalogLoaderForContentKind,
} from './gameServerTypes';

describe('gameServerTypes', () => {
  it('lists all types from groups', () => {
    expect(ALL_GAME_SERVER_TYPES).toContain('vanilla');
    expect(ALL_GAME_SERVER_TYPES).toContain('paper');
    expect(ALL_GAME_SERVER_TYPES).toContain('mohist');
    expect(ALL_GAME_SERVER_TYPES).toHaveLength(14);
    expect(ALL_GAME_SERVER_TYPES).toContain('velocity');
    expect(ALL_GAME_SERVER_TYPES).toContain('waterfall');
    expect(ALL_GAME_SERVER_TYPES).toContain('bungeecord');
  });

  it('recognizes known types', () => {
    expect(isKnownGameServerType('forge')).toBe(true);
    expect(isKnownGameServerType('arclight')).toBe(true);
    expect(isKnownGameServerType('unknown')).toBe(false);
  });

  it('defaults to vanilla', () => {
    expect(DEFAULT_GAME_SERVER_TYPE).toBe('vanilla');
  });

  it('reports plugin, mod, and datapack capabilities', () => {
    expect(gameServerSupportsPlugins('paper')).toBe(true);
    expect(gameServerSupportsMods('paper')).toBe(false);
    expect(gameServerSupportsPlugins('forge')).toBe(false);
    expect(gameServerSupportsMods('forge')).toBe(true);
    expect(gameServerTypeCapabilities('mohist')).toEqual({
      plugins: true,
      mods: true,
      datapacks: true,
      clientContent: true,
    });
    expect(gameServerTypeCapabilities('vanilla')).toEqual({
      plugins: false,
      mods: false,
      datapacks: true,
      clientContent: true,
    });
    expect(gameServerCatalogTabs('forge')).toEqual(['mod', 'resourcepack', 'shader', 'datapack']);
    expect(gameServerCatalogTabs('vanilla')).toEqual(['resourcepack', 'shader', 'datapack']);
    expect(gameServerSupportsPlugins('velocity')).toBe(true);
    expect(gameServerSupportsMods('velocity')).toBe(false);
    expect(gameServerTypeCapabilities('velocity')).toEqual({
      plugins: true,
      mods: false,
      datapacks: false,
      clientContent: false,
    });
    expect(gameServerCatalogTabs('velocity')).toEqual([]);
    expect(gameServerSupportsPlugins('waterfall')).toBe(true);
    expect(gameServerTypeCapabilities('bungeecord')).toEqual({
      plugins: true,
      mods: false,
      datapacks: false,
      clientContent: false,
    });
  });

  it('formats labels for known and unknown types', () => {
    const t = (key: string) => (key === 'servers.gameServerType.paper' ? 'Paper' : key);
    expect(gameServerTypeLabelText(t, undefined)).toBe('—');
    expect(gameServerTypeLabelText(t, 'paper')).toBe('Paper');
    expect(gameServerTypeLabelText(t, 'custom-core')).toBe('custom-core');
  });

  it('uses datapack as the version loader for datapack catalogs', () => {
    expect(catalogLoaderForContentKind('datapack', 'paper')).toBe('datapack');
    expect(catalogLoaderForContentKind('plugin', 'paper')).toBe('paper');
    expect(catalogLoaderForContentKind('mod', 'forge')).toBe('forge');
    expect(catalogLoaderForContentKind('resourcepack', 'paper')).toBeUndefined();
  });

  it('limits velocity plugin catalogs to hangar and modrinth', () => {
    expect(pluginCatalogSources('velocity')).toEqual(['hangar', 'modrinth']);
    expect(pluginCatalogSources('paper')).toEqual(['hangar', 'spigot', 'bukkit', 'modrinth']);
    expect(pluginCatalogSources('waterfall')).toEqual(['hangar', 'spigot', 'bukkit', 'modrinth']);
  });
});
