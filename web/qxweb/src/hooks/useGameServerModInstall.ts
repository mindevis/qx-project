import { useCallback, useState } from 'react';
import {
  api,
  type GameServerContentKind,
  type GameServerContentSyncBody,
} from '@/api/client';
import type { ModInstallParams } from '@/hooks/useModInstall';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  contentKindHasSide,
  gameServerInstallSide,
  instanceResourceContentTarget,
} from '@/lib/modSync';

function syncContent(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
  body: GameServerContentSyncBody,
) {
  switch (kind) {
    case 'plugin':
      return api.syncPluginToGameServer(vpsId, gameServerId, body);
    case 'datapack':
      return api.syncDatapackToGameServer(vpsId, gameServerId, body);
    case 'resourcepack':
      return api.syncResourcepackToGameServer(vpsId, gameServerId, body);
    case 'shader':
      return api.syncShaderToGameServer(vpsId, gameServerId, body);
    default:
      return api.syncModToGameServer(vpsId, gameServerId, body);
  }
}

export function useGameServerModInstall(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
) {
  const { t } = useI18n();
  const message = useMessage();
  const [installingVersionId, setInstallingVersionId] = useState<string>();

  const installOne = useCallback(
    async (params: ModInstallParams): Promise<boolean> => {
      const file = params.version.files[0];
      if (!file?.url) {
        throw new Error(t('qxmods.install.noFile'));
      }
      const side = contentKindHasSide(kind) ? gameServerInstallSide(params.side) : undefined;
      await syncContent(kind, vpsId, gameServerId, {
        source: params.source,
        project_id: params.projectId,
        project_name: params.projectName,
        version_id: params.version.id,
        version_number: params.version.version_number,
        version_type: params.version.version_type,
        filename: file.filename,
        download_url: file.url,
        icon_url: params.iconUrl,
        downloads: params.downloads,
        file_size: params.fileSize ?? file.size,
        mod_target: contentKindHasSide(kind)
          ? instanceResourceContentTarget({ side_override: side, resource_type: kind })
          : undefined,
        side_override: side,
        replace_filename: params.replaceFilename,
      });
      return true;
    },
    [gameServerId, kind, t, vpsId],
  );

  const installBatch = useCallback(
    async (items: ModInstallParams[]) => {
      if (items.length === 0) return false;
      const primary = items[items.length - 1];
      setInstallingVersionId(primary.version.id);
      try {
        for (const item of items) {
          await installOne(item);
        }
        const replaced = items.some((item) => Boolean(item.replaceFilename));
        message.success(
          t(replaced ? 'gameServerDetail.content.versionUpdated' : 'gameServerDetail.content.installed'),
        );
        return true;
      } catch (e) {
        message.error(e instanceof Error ? e.message : t('qxmods.install.failed'));
        return false;
      } finally {
        setInstallingVersionId(undefined);
      }
    },
    [installOne, message, t],
  );

  return { installingVersionId, installBatch };
}
