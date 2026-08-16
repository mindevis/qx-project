import { type ReactNode } from 'react';
import { api, type GameServerContentKind, type ModProjectType, type ModTarget } from '@/api/client';
import { ModCatalogProvider } from '@/components/ModCatalogContext';
import { useGameServerModInstall } from '@/hooks/useGameServerModInstall';
import { pluginLoaderForServerType, type VpsGameServerType } from '@/lib/gameServerTypes';

function projectTypeForKind(kind: GameServerContentKind): ModProjectType {
  switch (kind) {
    case 'plugin':
      return 'plugin';
    case 'datapack':
      return 'datapack';
    default:
      return 'mod';
  }
}

export function GameServerCatalogProvider({
  kind,
  vpsId,
  gameServerId,
  serverType,
  mcVersion,
  modTarget,
  children,
}: {
  kind: GameServerContentKind;
  vpsId: string;
  gameServerId: string;
  serverType: VpsGameServerType;
  mcVersion: string;
  modTarget?: ModTarget;
  children: ReactNode;
}) {
  const { installingVersionId, installBatch } = useGameServerModInstall(
    kind,
    vpsId,
    gameServerId,
    modTarget,
  );
  const resourceType = projectTypeForKind(kind);
  const loader =
    kind === 'plugin' ? pluginLoaderForServerType(serverType) : kind === 'mod' ? serverType : '';

  return (
    <ModCatalogProvider
      value={{
        loader,
        mcVersion,
        basePath: `/servers/${vpsId}/game-servers/${gameServerId}/${kind}`,
        canSyncToServer: false,
        showServerOnlyModal: false,
        installingVersionId,
        listInstalled: async () => {
          const res = await api.listGameServerResources(vpsId, gameServerId, {
            kind: resourceType,
            mod_target: kind === 'mod' ? modTarget : undefined,
          });
          return res.items ?? [];
        },
        installBatch,
        uninstall: async ({ filename }) => {
          if (!filename) {
            throw new Error('filename required');
          }
          switch (kind) {
            case 'plugin':
              await api.deleteVpsGameServerPlugin(vpsId, gameServerId, { filename });
              break;
            case 'datapack':
              await api.deleteVpsGameServerDatapack(vpsId, gameServerId, { filename });
              break;
            default:
              await api.deleteVpsGameServerMod(vpsId, gameServerId, {
                filename,
                mod_target: modTarget,
              });
          }
        },
      }}
    >
      {children}
    </ModCatalogProvider>
  );
}
