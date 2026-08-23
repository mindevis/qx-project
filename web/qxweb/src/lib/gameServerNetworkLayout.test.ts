import { describe, expect, it } from 'vitest';
import {
  aliasFromServerName,
  assignNetworkMemberRole,
  canMoveNetworkMember,
  groupGameServersByNetwork,
  groupNetworkMembers,
  suggestedAliasForServer,
  suggestedRoleForServer,
} from './gameServerNetworkLayout';

describe('gameServerNetworkLayout', () => {
  it('slugs aliases and suggests roles', () => {
    expect(aliasFromServerName('Lobby #1')).toBe('lobby-1');
    expect(suggestedRoleForServer('velocity', [])).toBe('proxy');
    expect(suggestedRoleForServer('waterfall', [])).toBe('proxy');
    expect(suggestedRoleForServer('bungeecord', [])).toBe('proxy');
    expect(suggestedRoleForServer('paper', [])).toBe('lobby');
    expect(suggestedRoleForServer('paper', [{ role: 'lobby' }])).toBe('backend');
    expect(suggestedRoleForServer(undefined, [])).toBe('lobby');
    expect(suggestedAliasForServer('qrpg-world-proxy', 'proxy')).toBe('proxy');
    expect(suggestedAliasForServer('Lobby #1', 'lobby')).toBe('lobby');
    expect(suggestedAliasForServer('qrpg-world-survival', 'backend')).toBe('qrpg-world-survival');
  });

  it('groups members into proxy, try, and world columns', () => {
    const grouped = groupNetworkMembers([
      { game_server_id: 'v', role: 'proxy' as const, sort_order: 0 },
      { game_server_id: 's', role: 'backend' as const, sort_order: 2 },
      { game_server_id: 'l', role: 'lobby' as const, sort_order: 1 },
      { game_server_id: 'n', role: 'backend' as const, sort_order: 1 },
    ]);
    expect(grouped.proxy.map((item) => item.game_server_id)).toEqual(['v']);
    expect(grouped.lobby.map((item) => item.game_server_id)).toEqual(['l']);
    expect(grouped.backend.map((item) => item.game_server_id)).toEqual(['n', 's']);
  });

  it('moves a world onto try and demotes the previous lobby', () => {
    const next = assignNetworkMemberRole(
      [
        { game_server_id: 'v', role: 'proxy', alias: 'proxy' },
        { game_server_id: 'l', role: 'lobby', alias: 'lobby' },
        { game_server_id: 's', role: 'backend', alias: 'survival' },
      ],
      's',
      'lobby',
      (id) => (id === 's' ? 'Survival' : id === 'l' ? 'Lobby' : 'Velocity'),
    );
    expect(next.find((item) => item.game_server_id === 's')).toEqual({
      game_server_id: 's',
      role: 'lobby',
      alias: 'lobby',
    });
    expect(next.find((item) => item.game_server_id === 'l')?.role).toBe('backend');
    expect(next.find((item) => item.game_server_id === 'v')?.role).toBe('proxy');
  });

  it('does not move a proxy off its column or a world onto proxy', () => {
    expect(canMoveNetworkMember('proxy', 'lobby')).toBe(false);
    expect(canMoveNetworkMember('backend', 'proxy')).toBe(false);
    expect(canMoveNetworkMember('backend', 'lobby')).toBe(true);
    const members = [
      { game_server_id: 'v', role: 'proxy' as const, alias: 'proxy' },
      { game_server_id: 's', role: 'backend' as const, alias: 'survival' },
    ];
    expect(assignNetworkMemberRole(members, 'v', 'lobby', () => 'Velocity')).toEqual(members);
    expect(assignNetworkMemberRole(members, 's', 'proxy', () => 'Survival')).toEqual(members);
  });

  it('nests assigned game servers under their project and keeps the rest loose', () => {
    const groups = groupGameServersByNetwork(
      [
        { id: 'v', name: 'Velocity' },
        { id: 'l', name: 'Lobby' },
        { id: 'c', name: 'Creative' },
      ],
      [
        {
          id: 'net-1',
          name: 'Mini-games',
          members: [
            { game_server_id: 'l', sort_order: 1 },
            { game_server_id: 'v', sort_order: 0 },
            { game_server_id: 'missing', sort_order: 2 },
          ],
        },
        { id: 'net-empty', name: 'Empty', members: [] },
      ],
    );
    expect(groups).toEqual([
      {
        key: 'net-1',
        network: { id: 'net-1', name: 'Mini-games' },
        games: [
          { id: 'v', name: 'Velocity' },
          { id: 'l', name: 'Lobby' },
        ],
      },
      {
        key: 'ungrouped',
        network: null,
        games: [{ id: 'c', name: 'Creative' }],
      },
    ]);
  });
});
