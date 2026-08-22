import type { GameServerNetworkMember, GameServerNetworkRole } from '@/api/client';
import { isKnownGameServerType, isProxyGameServerType } from '@/lib/gameServerTypes';

export type NetworkDiagramNode = {
  id: string;
  kind: 'players' | 'server';
  x: number;
  y: number;
  width: number;
  height: number;
  label: string;
  subtitle?: string;
  role?: GameServerNetworkRole;
  href?: string;
  gameServerId?: string;
};

export type NetworkDiagramEdge = {
  id: string;
  from: string;
  to: string;
  kind: 'connect' | 'try' | 'transfer';
};

export type NetworkDiagramLayout = {
  width: number;
  height: number;
  nodes: NetworkDiagramNode[];
  edges: NetworkDiagramEdge[];
};

const NODE_W = 176;
const NODE_H = 72;
const PLAYERS_W = 140;
const PLAYERS_H = 48;
const ROW_GAP = 88;
const COL_GAP = 28;
const PAD_X = 24;
const PAD_Y = 16;

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
  return aliasFromServerName(name);
}

export function suggestedRoleForServer(
  serverType: string | undefined,
  existing: Array<{ role: GameServerNetworkRole }>,
): GameServerNetworkRole {
  if (
    serverType &&
    isKnownGameServerType(serverType) &&
    isProxyGameServerType(serverType)
  ) {
    return 'proxy';
  }
  if (!existing.some((item) => item.role === 'lobby')) return 'lobby';
  return 'backend';
}

export function layoutGameServerNetwork(
  members: GameServerNetworkMember[],
  labels: { players: string },
): NetworkDiagramLayout {
  const proxy = members.find((item) => item.role === 'proxy');
  const lobby = members.find((item) => item.role === 'lobby');
  const worlds = members
    .filter((item) => item.role === 'backend' || (item.role === 'lobby' && proxy))
    .sort((a, b) => a.sort_order - b.sort_order);

  const bottom = worlds.length > 0 ? worlds : members.filter((item) => item.role !== 'proxy');
  const cols = Math.max(bottom.length, 1);
  const contentW = cols * NODE_W + (cols - 1) * COL_GAP;
  const width = Math.max(contentW, NODE_W) + PAD_X * 2;

  const playersX = (width - PLAYERS_W) / 2;
  const playersY = PAD_Y;
  const proxyY = playersY + PLAYERS_H + ROW_GAP;
  const worldsY = proxy ? proxyY + NODE_H + ROW_GAP : playersY + PLAYERS_H + ROW_GAP;
  const worldsStartX = (width - (bottom.length * NODE_W + Math.max(bottom.length - 1, 0) * COL_GAP)) / 2;

  const nodes: NetworkDiagramNode[] = [
    {
      id: 'players',
      kind: 'players',
      x: playersX,
      y: playersY,
      width: PLAYERS_W,
      height: PLAYERS_H,
      label: labels.players,
    },
  ];
  const edges: NetworkDiagramEdge[] = [];

  if (proxy) {
    const proxyX = (width - NODE_W) / 2;
    nodes.push(serverNode(proxy, proxyX, proxyY));
    edges.push({ id: 'e-players-proxy', from: 'players', to: proxy.game_server_id, kind: 'connect' });
    bottom.forEach((member, index) => {
      const x = worldsStartX + index * (NODE_W + COL_GAP);
      if (!nodes.some((node) => node.id === member.game_server_id)) {
        nodes.push(serverNode(member, x, worldsY));
      }
      edges.push({
        id: `e-proxy-${member.game_server_id}`,
        from: proxy.game_server_id,
        to: member.game_server_id,
        kind: member.role === 'lobby' ? 'try' : 'connect',
      });
    });
    if (lobby) {
      for (const member of bottom) {
        if (member.game_server_id === lobby.game_server_id) continue;
        edges.push({
          id: `e-lobby-${member.game_server_id}`,
          from: lobby.game_server_id,
          to: member.game_server_id,
          kind: 'transfer',
        });
      }
    }
  } else {
    edges.push({
      id: 'e-players-first',
      from: 'players',
      to: bottom[0]?.game_server_id ?? members[0]?.game_server_id ?? 'players',
      kind: 'connect',
    });
    bottom.forEach((member, index) => {
      nodes.push(serverNode(member, worldsStartX + index * (NODE_W + COL_GAP), worldsY));
    });
  }

  const height =
    (proxy && bottom.length > 0 ? worldsY : proxy ? proxyY : worldsY) + NODE_H + PAD_Y;
  return { width, height, nodes, edges };
}

function serverNode(member: GameServerNetworkMember, x: number, y: number): NetworkDiagramNode {
  return {
    id: member.game_server_id,
    kind: 'server',
    x,
    y,
    width: NODE_W,
    height: NODE_H,
    label: member.name || member.alias,
    subtitle: `${member.alias} · :${member.port}`,
    role: member.role,
    gameServerId: member.game_server_id,
  };
}

export function nodeAnchor(
  node: NetworkDiagramNode,
  side: 'top' | 'bottom' | 'left' | 'right',
): { x: number; y: number } {
  switch (side) {
    case 'top':
      return { x: node.x + node.width / 2, y: node.y };
    case 'bottom':
      return { x: node.x + node.width / 2, y: node.y + node.height };
    case 'left':
      return { x: node.x, y: node.y + node.height / 2 };
    case 'right':
      return { x: node.x + node.width, y: node.y + node.height / 2 };
  }
}
