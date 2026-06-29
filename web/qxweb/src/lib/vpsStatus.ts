import type { GameServer } from '@/api/client';

export type VpsHostStatus = 'pending' | 'active' | 'error';

/** Dedicated server host record — not QXAgent and not a Minecraft game server. */
export function getVpsHostStatus(server: Pick<GameServer, 'status'>): VpsHostStatus {
  if (server.status === 'pending') {
    return 'pending';
  }
  if (server.status === 'error') {
    return 'error';
  }
  return 'active';
}

export function vpsHostStatusColor(status: VpsHostStatus): string {
  switch (status) {
    case 'active':
      return 'success';
    case 'error':
      return 'error';
    default:
      return 'default';
  }
}
