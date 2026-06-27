import { describe, expect, it, vi, afterEach } from 'vitest';
import { gameServerUpstreamUrl } from './gameServerUpstream';

describe('gameServerUpstream', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('uses production host outside dev', () => {
    vi.stubEnv('DEV', false);
    expect(gameServerUpstreamUrl('forge', '/net/minecraftforge/forge/promotions_slim.json')).toBe(
      'https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json',
    );
  });

  it('uses vite proxy prefix in dev', () => {
    vi.stubEnv('DEV', true);
    expect(gameServerUpstreamUrl('papermc', '/v2/projects/paper')).toBe(
      '/upstream/papermc/v2/projects/paper',
    );
  });
});
