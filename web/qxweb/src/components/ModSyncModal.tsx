import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Empty, Modal, Radio, Spin, Typography } from 'antd';
import {
  api,
  type ModCatalogItem,
  type ModVersion,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { gameServerSyncTargetKey, loadGameServerSyncTargets } from '@/lib/gameServerSyncTargets';
import { isModOnServer } from '@/lib/modSync';
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
  instanceLoader: string;
  instanceMcVersion?: string;
  onClose: () => void;
};

export function ModSyncModal({
  open,
  selection,
  instanceLoader,
  instanceMcVersion,
  onClose,
}: ModSyncModalProps) {
  const { t } = useI18n();
  const message = useMessage();
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
    if (!selection || !selectedTarget) return false;
    return isModOnServer(selectedTarget.serverMods, selection.version);
  }, [selectedTarget, selection]);

  const promptServerRestart = (target: (typeof targets)[number]) => {
    Modal.confirm({
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
    if (!selection || !selectedTarget) return;
    const file = selection.version.files[0];
    if (!file) {
      const messageText = t('qxmods.sync.noFile');
      setSyncFeedback({ kind: 'error', message: messageText });
      message.error(messageText);
      return;
    }
    const target = selectedTarget;
    setSyncing(true);
    setSyncFeedback(null);
    try {
      const res = await api.syncModToGameServer(target.vpsId, target.gameServer.id, {
        source: selection.source,
        project_id: selection.projectId,
        version_id: selection.version.id,
        filename: file.filename,
        download_url: file.url,
        project_name: selection.projectName,
        version_number: selection.version.version_number,
      });
      if (res.status === 'already_installed') {
        const messageText = t('qxmods.sync.alreadyOnServer');
        setSyncFeedback({ kind: 'info', message: messageText });
        message.info(messageText);
        return;
      }
      const messageText =
        res.status === 'installed' ? t('qxmods.sync.installed') : t('qxmods.sync.queued');
      setSyncFeedback({ kind: 'success', message: messageText });
      message.success(messageText);
      promptServerRestart(target);
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
                selection != null && isModOnServer(target.serverMods, selection.version);
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
