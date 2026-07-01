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

  const installVersion = useCallback(
    async (params: {
      source: ModSource;
      projectId: string;
      projectName: string;
      version: ModVersion;
      resourceType: ModProjectType;
    }) => {
      const file = params.version.files[0];
      if (!file) {
        message.error(t('qxmods.install.noFile'));
        return false;
      }
      setInstallingVersionId(params.version.id);
      setInstalledVersionId(undefined);
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
        });

        return await new Promise<boolean>((resolve) => {
          const finish = (ok: boolean) => {
            clearPoll();
            setInstallingVersionId(undefined);
            resolve(ok);
          };

          const poll = async () => {
            try {
              const req = await api.getModInstallRequest(created.id);
              if (!TERMINAL_STATUSES.has(req.status)) {
                return;
              }
              if (req.status === 'completed') {
                setInstalledVersionId(params.version.id);
                message.success(t('qxmods.install.completed'));
                finish(true);
                return;
              }
              message.error(
                req.error_code === 'INSTALL_FAILED'
                  ? t('qxmods.install.failed')
                  : t('qxmods.install.deviceRequired'),
              );
              finish(false);
            } catch (e) {
              message.error(e instanceof Error ? e.message : t('qxmods.install.failed'));
              finish(false);
            }
          };

          void poll();
          pollRef.current = setInterval(() => void poll(), POLL_MS);
        });
      } catch (e) {
        const msg = e instanceof Error ? e.message : t('qxmods.install.failed');
        if (msg.includes('device not linked') || msg.includes('FORBIDDEN')) {
          message.error(t('qxmods.install.deviceRequired'));
        } else {
          message.error(msg);
        }
        setInstallingVersionId(undefined);
        return false;
      }
    },
    [clearPoll, instanceId, message, t],
  );

  return {
    installingVersionId,
    installedVersionId,
    installVersion,
    resetInstalled: () => setInstalledVersionId(undefined),
  };
}
