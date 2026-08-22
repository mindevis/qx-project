import { describe, expect, it } from 'vitest';
import {
  aliasFromServerName,
  layoutGameServerNetwork,
  suggestedRoleForServer,
} from './gameServerNetworkLayout';
import type { GameServerNetworkMember } from '@/api/client';

function member(
  partial: Partial<GameServerNetworkMember> & Pick<GameServerNetworkMember, 'game_server_id' | 'role' | 'alias'>,
): GameServerNetworkMember {
  return {
    id: partial.id ?? partial.game_server_id,
    sort_order: partial.sort_order ?? 0,
    name: partial.name ?? partial.alias,
    server_type: partial.server_type ?? 'paper',
    port: partial.port ?? 25565,
    status: partial.status ?? 'stopped',
    ...partial,
  };
}

describe('gameServerNetworkLayout', () => {
  it('slugs aliases and suggests roles', () => {
    expect(aliasFromServerName('Lobby #1')).toBe('lobby-1');
    expect(suggestedRoleForServer('velocity', [])).toBe('proxy');
    expect(suggestedRoleForServer('waterfall', [])).toBe('proxy');
    expect(suggestedRoleForServer('bungeecord', [])).toBe('proxy');
    expect(suggestedRoleForServer('paper', [])).toBe('lobby');
    expect(suggestedRoleForServer('paper', [{ role: 'lobby' }])).toBe('backend');
  });

  it('lays out players -> velocity -> lobby and worlds', () => {
    const layout = layoutGameServerNetwork(
      [
        member({
          game_server_id: 'v',
          role: 'proxy',
          alias: 'proxy',
          name: 'Velocity',
          server_type: 'velocity',
          port: 25565,
        }),
        member({
          game_server_id: 'l',
          role: 'lobby',
          alias: 'lobby',
          name: 'Lobby',
          port: 25566,
          sort_order: 1,
        }),
        member({
          game_server_id: 's',
          role: 'backend',
          alias: 'survival',
          name: 'Survival',
          port: 25567,
          sort_order: 2,
        }),
      ],
      { players: 'Players' },
    );

    expect(layout.nodes.map((node) => node.id)).toEqual(['players', 'v', 'l', 's']);
    expect(layout.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ from: 'players', to: 'v', kind: 'connect' }),
        expect.objectContaining({ from: 'v', to: 'l', kind: 'try' }),
        expect.objectContaining({ from: 'v', to: 's', kind: 'connect' }),
        expect.objectContaining({ from: 'l', to: 's', kind: 'transfer' }),
      ]),
    );
  });
});
