import { api } from '@/api/client';
import {
  DEFAULT_GAME_SERVER_TYPE,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';

export type { VpsGameServerType } from '@/lib/gameServerTypes';

export const DEFAULT_MC_GAME_PORT = 25565;

export type VpsGameServerStatus =
  | 'installing'
  | 'starting'
  | 'running'
  | 'stopped'
  | 'error';

export type VpsGameServer = {
  id: string;
  name: string;
  server_type?: VpsGameServerType;
  mc_version?: string;
  loader_version?: string;
  address?: string;
  port?: number;
  rcon_password?: string;
  rcon_port?: number;
  status: VpsGameServerStatus;
  show_in_monitoring?: boolean;
  monitoring_description?: string;
  banner_url?: string;
  monitoring_tags?: string[];
  last_error?: string;
  created_at: string;
};

export type CreateVpsGameServerInput = {
  name: string;
  server_type?: VpsGameServerType;
  mc_version: string;
  loader_version?: string;
  address?: string;
  port?: number;
  show_in_monitoring?: boolean;
  monitoring_description?: string;
  banner_url?: string;
  monitoring_tags?: string[];
};

export type UpdateVpsGameServerInput = {
  name?: string;
  address?: string;
  port?: number;
  show_in_monitoring?: boolean;
  monitoring_description?: string;
  banner_url?: string;
  monitoring_tags?: string[];
};

function mapGameServer(item: {
  id: string;
  name: string;
  server_type: string;
  mc_version: string;
  loader_version?: string;
  address?: string;
  port: number;
  rcon_password?: string;
  rcon_port?: number;
  status: string;
  show_in_monitoring?: boolean;
  monitoring_description?: string;
  banner_url?: string;
  monitoring_tags?: string[];
  last_error?: string;
  created_at: string;
}): VpsGameServer {
  return {
    id: item.id,
    name: item.name,
    server_type: item.server_type as VpsGameServerType,
    mc_version: item.mc_version,
    loader_version: item.loader_version,
    address: item.address,
    port: item.port,
    rcon_password: item.rcon_password,
    rcon_port: item.rcon_port,
    status: item.status as VpsGameServerStatus,
    show_in_monitoring: item.show_in_monitoring,
    monitoring_description: item.monitoring_description,
    banner_url: item.banner_url,
    monitoring_tags: item.monitoring_tags,
    last_error: item.last_error,
    created_at: item.created_at,
  };
}

export function suggestDefaultGamePort(existing: VpsGameServer[]): number {
  const used = new Set(existing.map((item) => item.port).filter((port) => port != null));
  let port = DEFAULT_MC_GAME_PORT;
  while (used.has(port) && port < 65535) {
    port += 1;
  }
  return port;
}

export async function listVpsGameServers(vpsId: string): Promise<VpsGameServer[]> {
  const data = await api.listVpsGameServers(vpsId);
  return (data.items ?? []).map(mapGameServer);
}

export async function addVpsGameServer(
  vpsId: string,
  input: CreateVpsGameServerInput,
): Promise<VpsGameServer> {
  const created = await api.createVpsGameServer(vpsId, {
    name: input.name,
    server_type: input.server_type ?? DEFAULT_GAME_SERVER_TYPE,
    mc_version: input.mc_version,
    loader_version: input.loader_version,
    address: input.address,
    port: input.port,
    show_in_monitoring: input.show_in_monitoring,
    monitoring_description: input.monitoring_description,
    banner_url: input.banner_url,
    monitoring_tags: input.monitoring_tags,
  });
  return mapGameServer(created);
}

export async function updateVpsGameServer(
  vpsId: string,
  gameServerId: string,
  input: UpdateVpsGameServerInput,
): Promise<VpsGameServer> {
  const updated = await api.updateVpsGameServer(vpsId, gameServerId, input);
  return mapGameServer(updated);
}

export async function removeVpsGameServer(vpsId: string, id: string): Promise<void> {
  await api.deleteVpsGameServer(vpsId, id);
}

export async function cloneVpsGameServer(vpsId: string, id: string): Promise<VpsGameServer> {
  const created = await api.cloneVpsGameServer(vpsId, id);
  return mapGameServer(created);
}

export async function reinstallVpsGameServer(vpsId: string, id: string): Promise<VpsGameServer> {
  const updated = await api.reinstallVpsGameServer(vpsId, id);
  return mapGameServer(updated);
}

export async function startVpsGameServer(vpsId: string, id: string): Promise<VpsGameServer> {
  const updated = await api.startVpsGameServer(vpsId, id);
  return mapGameServer(updated);
}

export async function stopVpsGameServer(vpsId: string, id: string): Promise<VpsGameServer> {
  const updated = await api.stopVpsGameServer(vpsId, id);
  return mapGameServer(updated);
}

export async function restartVpsGameServer(vpsId: string, id: string): Promise<VpsGameServer> {
  const updated = await api.restartVpsGameServer(vpsId, id);
  return mapGameServer(updated);
}

export function isVpsGameServerProvisioning(status: VpsGameServerStatus): boolean {
  return status === 'installing' || status === 'starting';
}
