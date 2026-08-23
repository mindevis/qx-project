import { type ReactNode } from 'react';
import { api, type GameServerContentKind, type ModProjectType } from '@/api/client';
import { ModCatalogProvider } from '@/components/ModCatalogContext';
import { useGameServerModInstall } from '@/hooks/useGameServerModInstall';
import { catalogLoaderForContentKind, type VpsGameServerType } from '@/lib/gameServerTypes';
import { instanceResourceContentTarget, instanceResourceModTarget } from '@/lib/modSync';

function projectTypeForKind(kind: GameServerContentKind): ModProjectType {
  return kind;
}

export function GameServerCatalogProvider({
  kind,
  vpsId,
  gameServerId,
  serverType,
  mcVersion,
  children,
}: {
  kind: GameServerContentKind;
  vpsId: string;
  gameServerId: string;
  serverType: VpsGameServerType;
  mcVersion: string;
  children: ReactNode;
}) {
  const { installingVersionId, installBatch } = useGameServerModInstall(kind, vpsId, gameServerId);
  const resourceType = projectTypeForKind(kind);
  const loader = catalogLoaderForContentKind(kind, serverType) ?? '';

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
            case 'resourcepack':
            case 'shader': {
              const res = await api.listGameServerResources(vpsId, gameServerId, { kind });
              const match = (res.items ?? []).find(
                (item) => item.filename.toLowerCase() === filename.toLowerCase(),
              );
              const body = {
                filename,
                mod_target: match ? instanceResourceContentTarget(match) : undefined,
              };
              if (kind === 'resourcepack') {
                await api.deleteVpsGameServerResourcepack(vpsId, gameServerId, body);
              } else {
                await api.deleteVpsGameServerShader(vpsId, gameServerId, body);
              }
              break;
            }
            default: {
              const res = await api.listGameServerResources(vpsId, gameServerId, { kind: 'mod' });
              const match = (res.items ?? []).find(
                (item) => item.filename.toLowerCase() === filename.toLowerCase(),
              );
              await api.deleteVpsGameServerMod(vpsId, gameServerId, {
                filename,
                mod_target: match ? instanceResourceModTarget(match) : undefined,
              });
            }
          }
        },
      }}
    >
      {children}
    </ModCatalogProvider>
  );
}
