import { useCallback, useState } from 'react';
import {
  api,
  type GameServerContentKind,
  type GameServerContentSyncBody,
  type ModTarget,
} from '@/api/client';
import type { ModInstallParams } from '@/hooks/useModInstall';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

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
    default:
      return api.syncModToGameServer(vpsId, gameServerId, body);
  }
}

export function useGameServerModInstall(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
  modTarget?: ModTarget,
) {
  const { t } = useI18n();
  const message = useMessage();
  const [installingVersionId, setInstallingVersionId] = useState<string>();

  const installOne = useCallback(
    async (params: ModInstallParams): Promise<boolean> => {
      const file = params.version.files[0];
      if (!file?.url) {
        message.error(t('qxmods.install.noFile'));
        return false;
      }
      try {
        const res = await syncContent(kind, vpsId, gameServerId, {
          source: params.source,
          project_id: params.projectId,
          project_name: params.projectName,
          version_id: params.version.id,
          version_number: params.version.version_number,
          filename: file.filename,
          download_url: file.url,
          icon_url: params.iconUrl,
          downloads: params.downloads,
          file_size: params.fileSize ?? file.size,
          mod_target: kind === 'mod' ? modTarget : undefined,
        });
        if (res.status === 'already_installed') {
          return true;
        }
        return true;
      } catch (e) {
        message.error(e instanceof Error ? e.message : t('qxmods.install.failed'));
        return false;
      }
    },
    [gameServerId, kind, message, modTarget, t, vpsId],
  );

  const installBatch = useCallback(
    async (items: ModInstallParams[]) => {
      if (items.length === 0) return false;
      const primary = items[items.length - 1];
      setInstallingVersionId(primary.version.id);
      let allOk = true;
      for (const item of items) {
        const ok = await installOne(item);
        if (!ok) {
          allOk = false;
          break;
        }
      }
      setInstallingVersionId(undefined);
      if (allOk) {
        message.success(t('gameServerDetail.content.installed'));
      } else {
        message.error(t('qxmods.install.failed'));
      }
      return allOk;
    },
    [installOne, message, t],
  );

  return { installingVersionId, installBatch };
}
