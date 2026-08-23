import type { GameServerNetworkRole } from '@/api/client';
import { isKnownGameServerType, isProxyGameServerType } from '@/lib/gameServerTypes';

export type NetworkMemberDraft = {
  game_server_id: string;
  role: GameServerNetworkRole;
  alias: string;
};

export function aliasFromServerName(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 63);
  return slug || 'server';
}

export function suggestedAliasForServer(name: string, role: GameServerNetworkRole): string {
  if (role === 'proxy') return 'proxy';
  if (role === 'lobby') return 'lobby';
  return aliasFromServerName(name);
}

export function suggestedRoleForServer(
  serverType: string | undefined,
  existing: Array<{ role: GameServerNetworkRole }>,
): GameServerNetworkRole {
  if (serverType && isKnownGameServerType(serverType) && isProxyGameServerType(serverType)) {
    return 'proxy';
  }
  if (!existing.some((item) => item.role === 'lobby')) return 'lobby';
  return 'backend';
}

export function groupNetworkMembers<T extends { role: GameServerNetworkRole; sort_order?: number }>(
  members: T[],
): { proxy: T[]; lobby: T[]; backend: T[] } {
  return {
    proxy: members.filter((item) => item.role === 'proxy'),
    lobby: members.filter((item) => item.role === 'lobby'),
    backend: members
      .filter((item) => item.role === 'backend')
      .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0)),
  };
}

export function canMoveNetworkMember(
  currentRole: GameServerNetworkRole,
  targetRole: GameServerNetworkRole,
): boolean {
  if (currentRole === targetRole) return false;
  if (currentRole === 'proxy') return false;
  return targetRole !== 'proxy';
}

export type GameServerNetworkGroup<T extends { id: string }> = {
  key: string;
  network: { id: string; name: string } | null;
  games: T[];
};

export function groupGameServersByNetwork<T extends { id: string }>(
  games: T[],
  networks: Array<{
    id: string;
    name: string;
    members?: Array<{ game_server_id: string; sort_order?: number }>;
  }>,
): GameServerNetworkGroup<T>[] {
  const byId = new Map(games.map((game) => [game.id, game]));
  const assigned = new Set<string>();
  const groups: GameServerNetworkGroup<T>[] = [];

  for (const network of networks) {
    const groupedGames = [...(network.members ?? [])]
      .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0))
      .map((member) => byId.get(member.game_server_id))
      .filter((game): game is T => game != null);
    if (groupedGames.length === 0) continue;
    for (const game of groupedGames) assigned.add(game.id);
    groups.push({
      key: network.id,
      network: { id: network.id, name: network.name },
      games: groupedGames,
    });
  }

  const ungrouped = games.filter((game) => !assigned.has(game.id));
  if (ungrouped.length > 0) {
    groups.push({ key: 'ungrouped', network: null, games: ungrouped });
  }
  return groups;
}

export function assignNetworkMemberRole(
  members: NetworkMemberDraft[],
  gameServerId: string,
  role: GameServerNetworkRole,
  resolveName: (gameServerId: string) => string,
): NetworkMemberDraft[] {
  const current = members.find((item) => item.game_server_id === gameServerId);
  if (!current || !canMoveNetworkMember(current.role, role)) return members;
  return members.map((item) => {
    if (item.game_server_id === gameServerId) {
      return {
        ...item,
        role,
        alias: suggestedAliasForServer(resolveName(item.game_server_id), role),
      };
    }
    if (role === 'lobby' && item.role === 'lobby') {
      return {
        ...item,
        role: 'backend',
        alias: suggestedAliasForServer(resolveName(item.game_server_id), 'backend'),
      };
    }
    return item;
  });
}
