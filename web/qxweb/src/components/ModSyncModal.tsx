import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Empty, Modal, Radio, Spin, Typography } from 'antd';
import {
  api,
  type ModCatalogItem,
  type ModVersion,
  type InstanceResource,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { useModal } from '@/hooks/useModal';
import { gameServerSyncTargetKey, loadGameServerSyncTargets } from '@/lib/gameServerSyncTargets';
import {
  isFilenameOnServer,
  isModOnServer,
  needsServerRestartAfterSync,
  resolveModSyncBodies,
  applyModTargetToBodies,
  type ModTarget,
} from '@/lib/modSync';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';

const { Text } = Typography;

type SyncFeedback = {
  kind: 'success' | 'error' | 'info';
  message: string;
};

export type ModSyncSelection = {
  source: ModCatalogItem['source'];
  projectId: string;
  projectName: string;
  version: ModVersion;
};

type ModSyncModalProps = {
  open: boolean;
  selection: ModSyncSelection | null;
  uploadedResource?: InstanceResource | null;
  instanceId?: string;
  instanceLoader: string;
  instanceMcVersion?: string;
  installedResources?: InstanceResource[];
  modTarget?: ModTarget;
  resourceType?: 'mod' | 'resourcepack' | 'shader';
  onClose: () => void;
};

export function ModSyncModal({
  open,
  selection,
  uploadedResource = null,
  instanceId,
  instanceLoader,
  instanceMcVersion,
  installedResources = [],
  modTarget = 'mods',
  resourceType = 'mod',
  onClose,
}: ModSyncModalProps) {
  const { t } = useI18n();
  const message = useMessage();
  const modal = useModal();
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [syncFeedback, setSyncFeedback] = useState<SyncFeedback | null>(null);
  const [targets, setTargets] = useState<Awaited<ReturnType<typeof loadGameServerSyncTargets>>>([]);
  const [selectedKey, setSelectedKey] = useState<string>();

  const loadTargets = useCallback(async () => {
    if (!open) return;
    setLoading(true);
    try {
      const loaded = await loadGameServerSyncTargets(instanceLoader, instanceMcVersion);
      setTargets(loaded);
      setSelectedKey(loaded[0] ? gameServerSyncTargetKey(loaded[0]) : undefined);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.sync.loadFailed'));
      setTargets([]);
    } finally {
      setLoading(false);
    }
  }, [instanceLoader, instanceMcVersion, message, open, t]);

  useEffect(() => {
    if (open) {
      setSyncFeedback(null);
      void loadTargets();
    }
  }, [loadTargets, open]);

  const selectedTarget = useMemo(
    () => targets.find((item) => gameServerSyncTargetKey(item) === selectedKey),
    [selectedKey, targets],
  );

  const alreadyOnServer = useMemo(() => {
    if (!selectedTarget) return false;
    if (uploadedResource) {
      return isFilenameOnServer(selectedTarget.serverMods, uploadedResource.filename);
    }
    if (!selection) return false;
    return isModOnServer(selectedTarget.serverMods, selection.version);
  }, [selectedTarget, selection, uploadedResource]);

  const promptServerRestart = (target: (typeof targets)[number]) => {
    modal.confirm({
      title: t('qxmods.sync.restartTitle'),
      content: t('qxmods.sync.restartPrompt', { name: target.gameServer.name }),
      okText: t('qxmods.sync.restartConfirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await restartVpsGameServer(target.vpsId, target.gameServer.id);
          message.success(t('servers.gameServerRestartStarted'));
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('common.error'));
          throw e;
        }
      },
    });
  };

  const handleSync = async () => {
    if (!selectedTarget) return;
    const target = selectedTarget;
    setSyncing(true);
    setSyncFeedback(null);
    try {
      if (uploadedResource) {
        if (!instanceId) {
          throw new Error(t('qxmods.sync.failed'));
        }
        const res = await api.syncUploadedInstanceResource(instanceId, {
          vps_id: target.vpsId,
          game_server_id: target.gameServer.id,
          filename: uploadedResource.filename,
          resource_type: uploadedResource.resource_type,
          mod_target: modTarget,
        });
        if (res.status === 'already_installed') {
          message.info(t('qxmods.sync.alreadyOnServer'));
          onClose();
          return;
        }
        message.success(
          res.status === 'installed' ? t('qxmods.sync.installed') : t('qxmods.sync.queued'),
        );
        if (needsServerRestartAfterSync(modTarget)) {
          promptServerRestart(target);
        }
        onClose();
        return;
      }

      if (!selection) return;
      const bodies = applyModTargetToBodies(
        await resolveModSyncBodies(
          selection,
          installedResources,
          instanceLoader,
          instanceMcVersion,
        ),
        modTarget,
      );
      let lastStatus: 'queued' | 'already_installed' | 'installed' = 'queued';
      const syncToServer = (body: Parameters<typeof api.syncModToGameServer>[2]) => {
        if (resourceType === 'resourcepack') {
          return api.syncResourcepackToGameServer(target.vpsId, target.gameServer.id, body);
        }
        if (resourceType === 'shader') {
          return api.syncShaderToGameServer(target.vpsId, target.gameServer.id, body);
        }
        return api.syncModToGameServer(target.vpsId, target.gameServer.id, body);
      };
      for (const body of bodies) {
        const res = await syncToServer(body);
        lastStatus = res.status;
      }
      if (lastStatus === 'already_installed' && bodies.length === 1) {
        message.info(t('qxmods.sync.alreadyOnServer'));
        onClose();
        return;
      }
      const messageText =
        lastStatus === 'installed' ? t('qxmods.sync.installed') : t('qxmods.sync.queued');
      message.success(
        bodies.length > 1
          ? t('qxmods.sync.installedWithDeps', { count: bodies.length })
          : messageText,
      );
      if (needsServerRestartAfterSync(modTarget)) {
        promptServerRestart(target);
      }
      onClose();
    } catch (e) {
      const messageText = e instanceof Error ? e.message : t('qxmods.sync.failed');
      setSyncFeedback({ kind: 'error', message: messageText });
      message.error(messageText);
    } finally {
      setSyncing(false);
    }
  };

  return (
    <Modal
      {...modalMotionProps}
      title={t('qxmods.sync.title')}
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose}>
          {t('common.cancel')}
        </Button>,
        <Button
          key="sync"
          type="primary"
          loading={syncing}
          disabled={!selectedTarget || alreadyOnServer || loading || syncFeedback?.kind === 'success'}
          onClick={() => void handleSync()}
        >
          {syncing
            ? t('qxmods.sync.syncing')
            : alreadyOnServer
              ? t('qxmods.sync.alreadyOnServer')
              : t('qxmods.sync.action')}
        </Button>,
      ]}
    >
      {syncFeedback ? (
        <Alert
          className="qxmods-sync-feedback"
          type={syncFeedback.kind}
          showIcon
          message={syncFeedback.message}
          style={{ marginBottom: 12 }}
        />
      ) : null}
      {loading ? (
        <div className="qxmods-sync-loading">
          <Spin />
        </div>
      ) : targets.length === 0 ? (
        <Empty description={t('qxmods.sync.noServers')} />
      ) : (
        <>
          <Text type="secondary" className="qxmods-sync-hint">
            {t('qxmods.sync.hint')}
          </Text>
          <Radio.Group
            className="qxmods-sync-list"
            value={selectedKey}
            onChange={(e) => setSelectedKey(e.target.value as string)}
          >
            {targets.map((target) => {
              const key = gameServerSyncTargetKey(target);
              const onServer =
                uploadedResource != null
                  ? isFilenameOnServer(target.serverMods, uploadedResource.filename)
                  : selection != null && isModOnServer(target.serverMods, selection.version);
              return (
                <Radio key={key} value={key} disabled={onServer} className="qxmods-sync-option">
                  <span className="qxmods-sync-option-name">{target.gameServer.name}</span>
                  <Text type="secondary" className="qxmods-sync-option-meta">
                    {target.vpsName}
                    {onServer ? ` · ${t('qxmods.sync.alreadyOnServer')}` : ''}
                  </Text>
                </Radio>
              );
            })}
          </Radio.Group>
        </>
      )}
    </Modal>
  );
}
