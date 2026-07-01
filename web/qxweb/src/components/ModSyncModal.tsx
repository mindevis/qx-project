import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Empty, Modal, Radio, Spin, Typography } from 'antd';
import {
  api,
  type GameServerFileEntry,
  type ModCatalogItem,
  type ModVersion,
  type VpsGameServerInstance,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { gameServerSupportsMods, isKnownGameServerType } from '@/lib/gameServerTypes';
import { isModOnServer } from '@/lib/modSync';
import { modalMotionProps } from '@/lib/modal';
import { listVpsGameServers, restartVpsGameServer, type VpsGameServer } from '@/lib/vpsGameServers';

const { Text } = Typography;

export type ModSyncSelection = {
  source: ModCatalogItem['source'];
  projectId: string;
  projectName: string;
  version: ModVersion;
};

type SyncTarget = {
  vpsId: string;
  vpsName: string;
  gameServer: VpsGameServer | VpsGameServerInstance;
  serverMods: GameServerFileEntry[];
};

type ModSyncModalProps = {
  open: boolean;
  selection: ModSyncSelection | null;
  instanceLoader: string;
  onClose: () => void;
};

export function ModSyncModal({ open, selection, instanceLoader, onClose }: ModSyncModalProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [targets, setTargets] = useState<SyncTarget[]>([]);
  const [selectedKey, setSelectedKey] = useState<string>();

  const loadTargets = useCallback(async () => {
    if (!open) return;
    setLoading(true);
    try {
      const serversRes = await api.listServers();
      const vpsList = serversRes.items ?? [];
      const loaded: SyncTarget[] = [];
      for (const vps of vpsList) {
        if (!vps.agent_online) continue;
        const gameServers = await listVpsGameServers(vps.id);
        for (const gs of gameServers) {
          const serverType = gs.server_type ?? 'vanilla';
          if (!isKnownGameServerType(serverType) || !gameServerSupportsMods(serverType)) {
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
          try {
            const modsRes = await api.listVpsGameServerMods(vps.id, gs.id);
            serverMods = modsRes.items ?? [];
          } catch {
            serverMods = [];
          }
          loaded.push({
            vpsId: vps.id,
            vpsName: vps.name || vps.slug,
            gameServer: gs,
            serverMods,
          });
        }
      }
      setTargets(loaded);
      setSelectedKey(loaded[0] ? `${loaded[0].vpsId}:${loaded[0].gameServer.id}` : undefined);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.sync.loadFailed'));
      setTargets([]);
    } finally {
      setLoading(false);
    }
  }, [instanceLoader, message, open, t]);

  useEffect(() => {
    if (open) {
      void loadTargets();
    }
  }, [loadTargets, open]);

  const selectedTarget = useMemo(
    () => targets.find((item) => `${item.vpsId}:${item.gameServer.id}` === selectedKey),
    [selectedKey, targets],
  );

  const alreadyOnServer = useMemo(() => {
    if (!selection || !selectedTarget) return false;
    return isModOnServer(selectedTarget.serverMods, selection.version);
  }, [selectedTarget, selection]);

  const promptServerRestart = (target: SyncTarget) => {
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
      message.error(t('qxmods.sync.noFile'));
      return;
    }
    const target = selectedTarget;
    setSyncing(true);
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
        message.info(t('qxmods.sync.alreadyOnServer'));
        onClose();
        return;
      }
      message.success(
        res.status === 'installed' ? t('qxmods.sync.installed') : t('qxmods.sync.queued'),
      );
      onClose();
      promptServerRestart(target);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.sync.failed'));
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
          disabled={!selectedTarget || alreadyOnServer || loading}
          onClick={() => void handleSync()}
        >
          {alreadyOnServer ? t('qxmods.sync.alreadyOnServer') : t('qxmods.sync.action')}
        </Button>,
      ]}
    >
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
              const key = `${target.vpsId}:${target.gameServer.id}`;
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
