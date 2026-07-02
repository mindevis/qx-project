import { api, type GameServerFileEntry } from '@/api/client';
import { gameServerSupportsMods, isKnownGameServerType } from '@/lib/gameServerTypes';
import { listVpsGameServers, type VpsGameServer } from '@/lib/vpsGameServers';

export type GameServerSyncTarget = {
  vpsId: string;
  vpsName: string;
  gameServer: VpsGameServer;
  serverMods: GameServerFileEntry[];
  clientMods?: GameServerFileEntry[];
};

export function gameServerSyncTargetKey(target: Pick<GameServerSyncTarget, 'vpsId' | 'gameServer'>) {
  return `${target.vpsId}:${target.gameServer.id}`;
}

export async function loadGameServerSyncTargets(
  instanceLoader: string,
  instanceMcVersion?: string,
): Promise<GameServerSyncTarget[]> {
  const serversRes = await api.listServers();
  const vpsList = serversRes.items ?? [];
  const loaded: GameServerSyncTarget[] = [];

  for (const vps of vpsList) {
    if (!vps.agent_online) continue;
    const gameServers = await listVpsGameServers(vps.id);
    for (const gs of gameServers) {
      const serverType = gs.server_type ?? 'vanilla';
      if (!isKnownGameServerType(serverType) || !gameServerSupportsMods(serverType)) {
        continue;
      }
      if (instanceMcVersion && gs.mc_version && gs.mc_version !== instanceMcVersion) {
        continue;
      }
      if (
        instanceLoader &&
        serverType !== instanceLoader &&
        !['mohist', 'magma', 'arclight'].includes(serverType)
      ) {
        continue;
      }
      let serverMods: GameServerFileEntry[] = [];
      let clientMods: GameServerFileEntry[] = [];
      try {
        const modsRes = await api.listVpsGameServerMods(vps.id, gs.id);
        serverMods = modsRes.items ?? [];
      } catch {
        serverMods = [];
      }
      try {
        const clientRes = await api.listVpsGameServerClientMods(vps.id, gs.id);
        clientMods = clientRes.items ?? [];
      } catch {
        clientMods = [];
      }
      loaded.push({
        vpsId: vps.id,
        vpsName: vps.name || vps.slug,
        gameServer: gs,
        serverMods,
        clientMods,
      });
    }
  }

  return loaded;
}
