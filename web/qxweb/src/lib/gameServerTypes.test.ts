import { describe, expect, it } from 'vitest';
import {
  ALL_GAME_SERVER_TYPES,
  DEFAULT_GAME_SERVER_TYPE,
  gameServerSupportsMods,
  gameServerSupportsPlugins,
  gameServerTypeCapabilities,
  isKnownGameServerType,
} from './gameServerTypes';

describe('gameServerTypes', () => {
  it('lists all types from groups', () => {
    expect(ALL_GAME_SERVER_TYPES).toContain('vanilla');
    expect(ALL_GAME_SERVER_TYPES).toContain('paper');
    expect(ALL_GAME_SERVER_TYPES).toContain('mohist');
    expect(ALL_GAME_SERVER_TYPES).toHaveLength(11);
  });

  it('recognizes known types', () => {
    expect(isKnownGameServerType('forge')).toBe(true);
    expect(isKnownGameServerType('arclight')).toBe(true);
    expect(isKnownGameServerType('unknown')).toBe(false);
  });

  it('defaults to vanilla', () => {
    expect(DEFAULT_GAME_SERVER_TYPE).toBe('vanilla');
  });

  it('reports plugin and mod capabilities', () => {
    expect(gameServerSupportsPlugins('paper')).toBe(true);
    expect(gameServerSupportsMods('paper')).toBe(false);
    expect(gameServerSupportsPlugins('forge')).toBe(false);
    expect(gameServerSupportsMods('forge')).toBe(true);
    expect(gameServerTypeCapabilities('mohist')).toEqual({ plugins: true, mods: true });
    expect(gameServerTypeCapabilities('vanilla')).toEqual({ plugins: false, mods: false });
  });
});
