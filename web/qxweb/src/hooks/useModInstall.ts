import { useCallback, useEffect, useRef, useState } from 'react';
import {
  api,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

const TERMINAL_STATUSES = new Set(['completed', 'failed', 'expired']);
const POLL_MS = 1500;
const INSTALL_TIMEOUT_MS = 5 * 60 * 1000;

export type ModInstallParams = {
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  resourceType: ModProjectType;
  iconUrl?: string;
  downloads?: number;
  fileSize?: number;
};

export function useModInstall(instanceId: string) {
  const { t } = useI18n();
  const message = useMessage();
  const [installingVersionId, setInstallingVersionId] = useState<string>();
  const [installedVersionId, setInstalledVersionId] = useState<string>();
  const pollRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  const clearPoll = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = undefined;
    }
  }, []);

  useEffect(() => clearPoll, [clearPoll]);

  const waitForInstall = useCallback(
    (requestId: string, versionId: string) =>
      new Promise<boolean>((resolve) => {
        const startedAt = Date.now();
        const finish = (ok: boolean) => {
          clearPoll();
          resolve(ok);
        };

        const poll = async () => {
          if (Date.now()-startedAt > INSTALL_TIMEOUT_MS) {
            finish(false);
            return;
          }
          try {
            const req = await api.getModInstallRequest(requestId);
            if (!TERMINAL_STATUSES.has(req.status)) {
              return;
            }
            if (req.status === 'completed') {
              setInstalledVersionId(versionId);
              finish(true);
              return;
            }
            finish(false);
          } catch {
            finish(false);
          }
        };

        void poll();
        pollRef.current = setInterval(() => void poll(), POLL_MS);
      }),
    [clearPoll],
  );

  const installOne = useCallback(
    async (params: ModInstallParams): Promise<boolean> => {
      const file = params.version.files[0];
      if (!file?.url) {
        message.error(t('qxmods.install.noFile'));
        return false;
      }
      try {
        const created = await api.createModInstallRequest({
          instance_id: instanceId,
          source: params.source,
          project_id: params.projectId,
          project_name: params.projectName,
          version_id: params.version.id,
          version_number: params.version.version_number,
          filename: file.filename,
          download_url: file.url,
          resource_type: params.resourceType,
          icon_url: params.iconUrl,
          downloads: params.downloads,
          file_size: params.fileSize ?? file.size,
        });
        return await waitForInstall(created.id, params.version.id);
      } catch (e) {
        const msg = e instanceof Error ? e.message : t('qxmods.install.failed');
        if (msg.includes('device not linked') || msg.includes('FORBIDDEN')) {
          message.error(t('qxmods.install.deviceRequired'));
        } else {
          message.error(msg);
        }
        return false;
      }
    },
    [instanceId, message, t, waitForInstall],
  );

  const installVersion = useCallback(
    async (params: ModInstallParams) => {
      setInstallingVersionId(params.version.id);
      setInstalledVersionId(undefined);
      const ok = await installOne(params);
      setInstallingVersionId(undefined);
      if (ok) {
        message.success(t('qxmods.install.completed'));
      } else if (!installingVersionId) {
        message.error(t('qxmods.install.failed'));
      }
      return ok;
    },
    [installOne, installingVersionId, message, t],
  );

  const installBatch = useCallback(
    async (items: ModInstallParams[]) => {
      if (items.length === 0) return false;
      const primary = items[items.length - 1];
      setInstallingVersionId(primary.version.id);
      setInstalledVersionId(undefined);
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
        setInstalledVersionId(primary.version.id);
        message.success(t('qxmods.install.completed'));
      } else {
        message.error(t('qxmods.install.failed'));
      }
      return allOk;
    },
    [installOne, message, t],
  );

  return {
    installingVersionId,
    installedVersionId,
    installVersion,
    installBatch,
    resetInstalled: () => setInstalledVersionId(undefined),
  };
}
