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
  if (role === 'lobby') return 'lobby';
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
    .filter((item) => item.role === 'backend')
    .sort((a, b) => a.sort_order - b.sort_order);

  const cols = Math.max(worlds.length, 1);
  const contentW = cols * NODE_W + (cols - 1) * COL_GAP;
  const width = Math.max(contentW, NODE_W) + PAD_X * 2;
  const centerX = (width - NODE_W) / 2;
  const worldsStartX =
    (width - (worlds.length * NODE_W + Math.max(worlds.length - 1, 0) * COL_GAP)) / 2;

  const nodes: NetworkDiagramNode[] = [
    {
      id: 'players',
      kind: 'players',
      x: (width - PLAYERS_W) / 2,
      y: PAD_Y,
      width: PLAYERS_W,
      height: PLAYERS_H,
      label: labels.players,
    },
  ];
  const edges: NetworkDiagramEdge[] = [];
  let y = PAD_Y + PLAYERS_H + ROW_GAP;
  let bottom = PAD_Y + PLAYERS_H;

  if (proxy) {
    nodes.push(serverNode(proxy, centerX, y));
    edges.push({ id: 'e-players-proxy', from: 'players', to: proxy.game_server_id, kind: 'connect' });
    bottom = y + NODE_H;
    y = bottom + ROW_GAP;
  }

  if (lobby) {
    nodes.push(serverNode(lobby, centerX, y));
    if (proxy) {
      edges.push({
        id: 'e-proxy-lobby',
        from: proxy.game_server_id,
        to: lobby.game_server_id,
        kind: 'try',
      });
    } else {
      edges.push({
        id: 'e-players-lobby',
        from: 'players',
        to: lobby.game_server_id,
        kind: 'connect',
      });
    }
    bottom = y + NODE_H;
    y = bottom + ROW_GAP;
  }

  worlds.forEach((member, index) => {
    nodes.push(serverNode(member, worldsStartX + index * (NODE_W + COL_GAP), y));
    bottom = y + NODE_H;
  });

  const parent = lobby ?? proxy;
  if (parent) {
    for (const member of worlds) {
      edges.push({
        id: `e-${parent.role}-${member.game_server_id}`,
        from: parent.game_server_id,
        to: member.game_server_id,
        kind: lobby ? 'transfer' : 'connect',
      });
    }
  } else if (worlds.length > 0 || members.length > 0) {
    edges.push({
      id: 'e-players-first',
      from: 'players',
      to: worlds[0]?.game_server_id ?? members[0]?.game_server_id ?? 'players',
      kind: 'connect',
    });
  }

  return { width, height: bottom + PAD_Y, nodes, edges };
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
    subtitle: `:${member.port}`,
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
