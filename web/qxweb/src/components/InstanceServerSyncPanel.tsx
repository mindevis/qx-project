import { useCallback, useEffect, useMemo, useState } from 'react';
import { CloudSyncOutlined } from '@ant-design/icons';
import { Button, Modal, Select, Spin, Tag, Typography } from 'antd';
import { api, type InstanceResource } from '@/api/client';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  gameServerSyncTargetKey,
  loadGameServerSyncTargets,
  type GameServerSyncTarget,
} from '@/lib/gameServerSyncTargets';
import {
  instanceResourceSupportsServerSync,
  isInstanceResourceOnServer,
} from '@/lib/modSync';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';
import './InstanceServerSyncPanel.css';

const { Text } = Typography;

type InstanceServerSyncPanelProps = {
  items: InstanceResource[];
};

export function InstanceServerSyncPanel({ items }: InstanceServerSyncPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, canSync } = useInstanceMods();
  const [loadingTargets, setLoadingTargets] = useState(false);
  const [targets, setTargets] = useState<GameServerSyncTarget[]>([]);
  const [selectedKey, setSelectedKey] = useState<string>();
  const [syncing, setSyncing] = useState(false);

  const loadTargets = useCallback(async () => {
    if (!canSync) {
      setTargets([]);
      setSelectedKey(undefined);
      return;
    }
    setLoadingTargets(true);
    try {
      const loaded = await loadGameServerSyncTargets(instance.loader);
      setTargets(loaded);
      setSelectedKey((prev) => {
        if (prev && loaded.some((item) => gameServerSyncTargetKey(item) === prev)) {
          return prev;
        }
        return loaded[0] ? gameServerSyncTargetKey(loaded[0]) : undefined;
      });
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.sync.loadFailed'));
      setTargets([]);
      setSelectedKey(undefined);
    } finally {
      setLoadingTargets(false);
    }
  }, [canSync, instance.loader, message, t]);

  useEffect(() => {
    void loadTargets();
  }, [loadTargets]);

  const selectedTarget = useMemo(
    () => targets.find((item) => gameServerSyncTargetKey(item) === selectedKey),
    [selectedKey, targets],
  );

  const syncableItems = useMemo(
    () => items.filter(instanceResourceSupportsServerSync),
    [items],
  );

  const serverMods = selectedTarget?.serverMods ?? [];

  const pendingItems = useMemo(
    () => syncableItems.filter((item) => !isInstanceResourceOnServer(serverMods, item)),
    [serverMods, syncableItems],
  );

  const syncedCount = syncableItems.length - pendingItems.length;

  const promptServerRestart = (target: GameServerSyncTarget) => {
    Modal.confirm({
      ...modalMotionProps,
      title: t('qxmods.sync.restartTitle'),
      content: t('qxmods.sync.restartBulkPrompt', { name: target.gameServer.name }),
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
    if (!selectedTarget || pendingItems.length === 0) return;
    const target = selectedTarget;
    setSyncing(true);
    let queued = 0;
    let failed = 0;

    try {
      for (const resource of pendingItems) {
        if (!resource.project_id || !resource.version_id) continue;
        try {
          const version = await api.getModVersion(resource.source, resource.project_id, resource.version_id, {
            loader: instance.loader,
            mc_version: instance.mc_version,
          });
          const file = version.files[0];
          if (!file?.url) {
            failed += 1;
            continue;
          }
          const res = await api.syncModToGameServer(target.vpsId, target.gameServer.id, {
            source: resource.source,
            project_id: resource.project_id,
            version_id: resource.version_id,
            filename: file.filename,
            download_url: file.url,
            project_name: resource.project_name,
            version_number: resource.version_number,
          });
          if (res.status !== 'already_installed') {
            queued += 1;
          }
        } catch {
          failed += 1;
        }
      }

      try {
        const modsRes = await api.listVpsGameServerMods(target.vpsId, target.gameServer.id);
        const serverMods = modsRes.items ?? [];
        setTargets((prev) =>
          prev.map((item) =>
            gameServerSyncTargetKey(item) === gameServerSyncTargetKey(target)
              ? { ...item, serverMods }
              : item,
          ),
        );
      } catch (e) {
        message.error(e instanceof Error ? e.message : t('qxmods.sync.statusLoadFailed'));
      }

      if (queued > 0) {
        message.success(t('qxmods.sync.bulkQueued', { count: queued }));
        promptServerRestart(target);
      } else if (failed > 0) {
        message.error(t('qxmods.sync.bulkFailed'));
      } else {
        message.info(t('qxmods.sync.allOnServer'));
      }
    } finally {
      setSyncing(false);
    }
  };

  if (!canSync) {
    return (
      <div className="instance-server-sync-panel">
        <Text type="secondary">{t('qxmods.sync.signIn')}</Text>
      </div>
    );
  }

  const statusTag =
    syncableItems.length === 0 ? (
      <Tag>{t('qxmods.sync.noSyncableMods')}</Tag>
    ) : !selectedTarget ? (
      <Tag>{t('qxmods.sync.pickServer')}</Tag>
    ) : pendingItems.length === 0 ? (
      <Tag color="success">{t('qxmods.sync.allOnServer')}</Tag>
    ) : (
      <Tag color="warning">
        {t('qxmods.sync.pendingCount', {
          pending: pendingItems.length,
          total: syncableItems.length,
        })}
      </Tag>
    );

  return (
    <div className="instance-server-sync-panel" aria-label={t('qxmods.sync.panelAria')}>
      <div className="instance-server-sync-head">
        <div>
          <Text strong className="instance-server-sync-title">
            {t('qxmods.sync.panelTitle')}
          </Text>
          <Text type="secondary" className="instance-server-sync-hint">
            {t('qxmods.sync.panelHint')}
          </Text>
        </div>
        {loadingTargets ? <Spin size="small" /> : statusTag}
      </div>

      {loadingTargets ? (
        <div className="instance-server-sync-loading">
          <Spin size="small" />
        </div>
      ) : targets.length === 0 ? (
        <Text type="secondary" className="instance-server-sync-empty">
          {t('qxmods.sync.noServers')}
        </Text>
      ) : (
        <div className="instance-server-sync-actions">
          <Select
            showSearch
            className="instance-server-sync-select"
            placeholder={t('qxmods.sync.serverPlaceholder')}
            value={selectedKey}
            loading={loadingTargets}
            disabled={syncing}
            optionFilterProp="label"
            options={targets.map((target) => ({
              value: gameServerSyncTargetKey(target),
              label: `${target.gameServer.name} (${target.vpsName})`,
            }))}
            onChange={(value) => setSelectedKey(value)}
          />
          <Button
            type="primary"
            icon={<CloudSyncOutlined />}
            loading={syncing}
            disabled={
              !selectedTarget ||
              pendingItems.length === 0 ||
              syncableItems.length === 0 ||
              loadingTargets
            }
            onClick={() => void handleSync()}
          >
            {t('qxmods.sync.action')}
          </Button>
        </div>
      )}

      {selectedTarget && syncableItems.length > 0 ? (
        <Text type="secondary" className="instance-server-sync-summary">
          {t('qxmods.sync.summary', {
            synced: syncedCount,
            total: syncableItems.length,
            server: selectedTarget.gameServer.name,
          })}
        </Text>
      ) : null}
    </div>
  );
}
