import { describe, expect, it } from 'vitest';
import { gameServerUpstreamUrl } from './gameServerUpstream';

describe('gameServerUpstream', () => {
  it('uses same-origin upstream proxy path in all environments', () => {
    expect(gameServerUpstreamUrl('forge', '/net/minecraftforge/forge/promotions_slim.json')).toBe(
      '/upstream/forge/net/minecraftforge/forge/promotions_slim.json',
    );
    expect(gameServerUpstreamUrl('mavenforge', '/net/minecraftforge/forge/maven-metadata.xml')).toBe(
      '/upstream/mavenforge/net/minecraftforge/forge/maven-metadata.xml',
    );
    expect(gameServerUpstreamUrl('papermc', '/v3/projects/paper')).toBe(
      '/upstream/papermc/v3/projects/paper',
    );
    expect(gameServerUpstreamUrl('bungeecord', '/job/BungeeCord/api/json')).toBe(
      '/upstream/bungeecord/job/BungeeCord/api/json',
    );
  });
});
