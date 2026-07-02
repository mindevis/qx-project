import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Modal, Radio, Spin, Typography } from 'antd';
import { api, type ModSource, type ModVersion } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { gameServerSyncTargetKey, loadGameServerSyncTargets } from '@/lib/gameServerSyncTargets';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';

const { Text } = Typography;

type ServerOnlyInstallModalProps = {
  open: boolean;
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  projectType?: 'mod' | 'resourcepack' | 'shader' | 'datapack' | 'plugin' | 'modpack';
  instanceLoader: string;
  instanceMcVersion?: string;
  onClose: () => void;
  onInstallToInstance: () => void;
};

export function ServerOnlyInstallModal({
  open,
  source,
  projectId,
  projectName,
  version,
  projectType = 'mod',
  instanceLoader,
  instanceMcVersion,
  onClose,
  onInstallToInstance,
}: ServerOnlyInstallModalProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [loading, setLoading] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [destination, setDestination] = useState<'server' | 'instance'>('server');
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
    if (open) void loadTargets();
  }, [loadTargets, open]);

  const selectedTarget = useMemo(
    () => targets.find((item) => gameServerSyncTargetKey(item) === selectedKey),
    [selectedKey, targets],
  );

  const handleConfirm = async () => {
    if (destination === 'instance') {
      onInstallToInstance();
      onClose();
      return;
    }
    if (!selectedTarget) return;
    const file = version.files[0];
    if (!file?.url) {
      message.error(t('gameServerDetail.content.noFile'));
      return;
    }
    setInstalling(true);
    try {
      const syncBody = {
        source,
        project_id: projectId,
        version_id: version.id,
        filename: file.filename,
        download_url: file.url,
        project_name: projectName,
        version_number: version.version_number,
      };
      if (projectType === 'resourcepack') {
        await api.syncResourcepackToGameServer(selectedTarget.vpsId, selectedTarget.gameServer.id, syncBody);
      } else if (projectType === 'shader') {
        await api.syncShaderToGameServer(selectedTarget.vpsId, selectedTarget.gameServer.id, syncBody);
      } else {
        await api.syncModToGameServer(selectedTarget.vpsId, selectedTarget.gameServer.id, syncBody);
      }
      message.success(t('qxmods.serverOnlyInstall.installedOnServer'));
      Modal.confirm({
        title: t('qxmods.sync.restartTitle'),
        content: t('qxmods.sync.restartPrompt', { name: selectedTarget.gameServer.name }),
        okText: t('qxmods.sync.restartConfirm'),
        cancelText: t('common.cancel'),
        onOk: async () => {
          await restartVpsGameServer(selectedTarget.vpsId, selectedTarget.gameServer.id);
          message.success(t('servers.gameServerRestartStarted'));
        },
      });
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.serverOnlyInstall.failed'));
    } finally {
      setInstalling(false);
    }
  };

  return (
    <Modal
      {...modalMotionProps}
      title={t('qxmods.serverOnlyInstall.title')}
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose}>
          {t('common.cancel')}
        </Button>,
        <Button
          key="confirm"
          type="primary"
          loading={installing}
          disabled={destination === 'server' && (!selectedTarget || loading)}
          onClick={() => void handleConfirm()}
        >
          {destination === 'server'
            ? t('qxmods.serverOnlyInstall.installOnServer')
            : t('qxmods.serverOnlyInstall.installOnInstance')}
        </Button>,
      ]}
    >
      <Text type="secondary">{t('qxmods.serverOnlyInstall.hint', { name: projectName })}</Text>
      <Radio.Group
        className="qxmods-server-only-dest"
        value={destination}
        onChange={(e) => setDestination(e.target.value as 'server' | 'instance')}
      >
        <Radio value="server">{t('qxmods.serverOnlyInstall.toServer')}</Radio>
        <Radio value="instance">{t('qxmods.serverOnlyInstall.toInstance')}</Radio>
      </Radio.Group>
      {destination === 'server' ? (
        loading ? (
          <Spin />
        ) : targets.length === 0 ? (
          <Text type="secondary">{t('qxmods.sync.noServers')}</Text>
        ) : (
          <Radio.Group
            className="qxmods-sync-list"
            value={selectedKey}
            onChange={(e) => setSelectedKey(e.target.value as string)}
          >
            {targets.map((target) => {
              const key = gameServerSyncTargetKey(target);
              return (
                <Radio key={key} value={key} className="qxmods-sync-option">
                  <span className="qxmods-sync-option-name">{target.gameServer.name}</span>
                  <Text type="secondary" className="qxmods-sync-option-meta">
                    {target.vpsName}
                  </Text>
                </Radio>
              );
            })}
          </Radio.Group>
        )
      ) : null}
    </Modal>
  );
}
