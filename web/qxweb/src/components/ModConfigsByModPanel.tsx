import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Modal,
  Radio,
  Space,
  Spin,
  Table,
  Tabs,
  Typography,
} from 'antd';
import { ArrowLeftOutlined, CloudDownloadOutlined, CloudUploadOutlined, FileOutlined } from '@ant-design/icons';
import { api, type GameServerFileEntry, type InstanceResource } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { gameServerSyncTargetKey, loadGameServerSyncTargets } from '@/lib/gameServerSyncTargets';
import {
  groupConfigFilesByMod,
  listConfigPaths,
  type ModConfigMod,
} from '@/lib/modConfigDiscovery';
import { modalMotionProps } from '@/lib/modal';

type FileApi = {
  listDir: (path: string) => Promise<GameServerFileEntry[]>;
  readFile: (path: string) => Promise<string>;
  writeFile: (path: string, content: string) => Promise<void>;
};

export type ConfigSyncContext = {
  instanceId: string;
  instanceLoader: string;
  instanceMcVersion?: string;
  deviceLinked: boolean;
  vpsId?: string;
  gameServerId?: string;
  agentOnline?: boolean;
};

type ModConfigsByModPanelProps = {
  mode: 'instance' | 'server';
  available: boolean;
  mods: ModConfigMod[];
  fileApi: FileApi;
  configSync?: ConfigSyncContext;
};

function basename(path: string): string {
  const parts = path.split('/');
  return parts[parts.length - 1] ?? path;
}

export function ModConfigsByModPanel({
  mode,
  available,
  mods,
  fileApi,
  configSync,
}: ModConfigsByModPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [loading, setLoading] = useState(true);
  const [configPaths, setConfigPaths] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState<string>('other');
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [serverPickerOpen, setServerPickerOpen] = useState(false);
  const [serverTargets, setServerTargets] = useState<Awaited<ReturnType<typeof loadGameServerSyncTargets>>>([]);
  const [serverTargetsLoading, setServerTargetsLoading] = useState(false);
  const [selectedServerKey, setSelectedServerKey] = useState<string>();

  const grouped = useMemo(
    () => groupConfigFilesByMod(mods, configPaths),
    [configPaths, mods],
  );

  const tabItems = useMemo(() => {
    const items = grouped.groups.map((group) => ({
      key: group.mod.key,
      label: group.mod.label,
      paths: group.paths,
    }));
    items.push({
      key: 'other',
      label: t('qxmods.configSync.otherTab'),
      paths: grouped.other,
    });
    return items;
  }, [grouped, t]);

  const activePaths = useMemo(
    () => tabItems.find((item) => item.key === activeTab)?.paths ?? [],
    [activeTab, tabItems],
  );

  const loadConfigTree = useCallback(async () => {
    if (!available) {
      setConfigPaths([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const paths = await listConfigPaths(async (path) => {
        const res = await fileApi.listDir(path);
        return res;
      });
      setConfigPaths(paths);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.filesLoadFailed'));
      setConfigPaths([]);
    } finally {
      setLoading(false);
    }
  }, [available, fileApi, message, t]);

  useEffect(() => {
    void loadConfigTree();
  }, [loadConfigTree]);

  useEffect(() => {
    if (tabItems.length > 0 && !tabItems.some((item) => item.key === activeTab)) {
      setActiveTab(tabItems[0].key);
    }
  }, [activeTab, tabItems]);

  const openFile = async (path: string) => {
    setSelectedFile(path);
    setFileLoading(true);
    try {
      const content = await fileApi.readFile(path);
      setFileContent(content);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.fileReadFailed'));
      setSelectedFile(null);
    } finally {
      setFileLoading(false);
    }
  };

  const saveFile = async () => {
    if (!selectedFile) return;
    setSaving(true);
    try {
      await fileApi.writeFile(selectedFile, fileContent);
      message.success(t('gameServerDetail.fileSaved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  const canSyncToServer =
    mode === 'instance' &&
    configSync?.deviceLinked === true &&
    Boolean(configSync.instanceLoader);

  const canPullFromServer =
    mode === 'server' &&
    configSync?.deviceLinked === true &&
    Boolean(configSync.instanceId) &&
    configSync.agentOnline === true;

  const openServerPicker = async () => {
    if (!configSync) return;
    setServerPickerOpen(true);
    setServerTargetsLoading(true);
    try {
      const loaded = await loadGameServerSyncTargets(
        configSync.instanceLoader,
        configSync.instanceMcVersion,
      );
      setServerTargets(loaded);
      setSelectedServerKey(loaded[0] ? gameServerSyncTargetKey(loaded[0]) : undefined);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.sync.loadFailed'));
      setServerTargets([]);
    } finally {
      setServerTargetsLoading(false);
    }
  };

  const syncToServer = async () => {
    if (!selectedFile || !configSync) return;
    const target = serverTargets.find((item) => gameServerSyncTargetKey(item) === selectedServerKey);
    if (!target) {
      message.warning(t('qxmods.configSync.pickServer'));
      return;
    }
    setSyncing(true);
    try {
      const content = await fileApi.readFile(selectedFile);
      await api.writeVpsGameServerFile(target.vpsId, target.gameServer.id, selectedFile, content);
      message.success(t('qxmods.configSync.success'));
      setServerPickerOpen(false);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.configSync.failed'));
    } finally {
      setSyncing(false);
    }
  };

  const pullFromServer = async () => {
    if (!selectedFile || !configSync?.instanceId) return;
    setSyncing(true);
    try {
      const content = await fileApi.readFile(selectedFile);
      await api.writeInstanceFile(configSync.instanceId, selectedFile, content);
      setFileContent(content);
      message.success(t('qxmods.configSync.success'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.configSync.failed'));
    } finally {
      setSyncing(false);
    }
  };

  if (!available) {
    return (
      <Typography.Paragraph type="secondary">
        {mode === 'instance'
          ? t('launcher.instanceSettingsModsConfigNote')
          : t('servers.gameServersAgentRequired')}
      </Typography.Paragraph>
    );
  }

  if (selectedFile) {
    return (
      <div className="game-server-files-editor">
        <Space className="game-server-files-toolbar" wrap>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => {
              setSelectedFile(null);
              setFileContent('');
            }}
          >
            {t('gameServerDetail.backToFiles')}
          </Button>
          <Typography.Text code>{selectedFile}</Typography.Text>
          <Button type="primary" loading={saving} onClick={() => void saveFile()}>
            {t('common.save')}
          </Button>
          {canSyncToServer ? (
            <Button icon={<CloudUploadOutlined />} onClick={() => void openServerPicker()}>
              {t('qxmods.configSync.toServer')}
            </Button>
          ) : null}
          {canPullFromServer ? (
            <Button icon={<CloudDownloadOutlined />} loading={syncing} onClick={() => void pullFromServer()}>
              {t('qxmods.configSync.pullFromServer')}
            </Button>
          ) : null}
        </Space>
        {fileLoading ? (
          <div className="servers-loading">
            <Spin />
          </div>
        ) : (
          <Input.TextArea
            className="game-server-files-textarea"
            value={fileContent}
            rows={18}
            onChange={(e) => setFileContent(e.target.value)}
          />
        )}
        <Modal
          title={t('qxmods.configSync.toServer')}
          open={serverPickerOpen}
          onCancel={() => setServerPickerOpen(false)}
          onOk={() => void syncToServer()}
          confirmLoading={syncing}
          okText={t('qxmods.configSync.toServer')}
          cancelText={t('common.cancel')}
          {...modalMotionProps}
        >
          {serverTargetsLoading ? (
            <div className="servers-loading">
              <Spin />
            </div>
          ) : serverTargets.length === 0 ? (
            <Empty description={t('qxmods.sync.noServers')} />
          ) : (
            <Radio.Group
              value={selectedServerKey}
              onChange={(e) => setSelectedServerKey(e.target.value as string)}
              style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
            >
              {serverTargets.map((target) => (
                <Radio key={gameServerSyncTargetKey(target)} value={gameServerSyncTargetKey(target)}>
                  {target.vpsName} · {target.gameServer.name}
                </Radio>
              ))}
            </Radio.Group>
          )}
        </Modal>
      </div>
    );
  }

  return (
    <div className="game-server-files">
      {loading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : tabItems.length === 0 || (grouped.groups.length === 0 && grouped.other.length === 0) ? (
        <Empty description={t('qxmods.configSync.noConfigs')} />
      ) : (
        <>
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            items={tabItems.map((item) => ({ key: item.key, label: item.label }))}
          />
          {activePaths.length === 0 ? (
            <Empty description={t('qxmods.configSync.noConfigs')} />
          ) : (
            <Table
              className="game-server-files-table"
              rowKey="path"
              size="small"
              pagination={false}
              dataSource={activePaths.map((path) => ({ path, name: basename(path) }))}
              onRow={(row) => ({
                onClick: () => void openFile(row.path),
                className: 'game-server-files-row',
              })}
              columns={[
                {
                  title: t('gameServerDetail.fileName'),
                  key: 'name',
                  render: (_, row) => (
                    <Space>
                      <FileOutlined />
                      <span>{row.name}</span>
                    </Space>
                  ),
                },
              ]}
            />
          )}
        </>
      )}
    </div>
  );
}

export function instanceResourceToModConfig(resource: InstanceResource): ModConfigMod | null {
  if (resource.resource_type !== 'mod') return null;
  const key = resource.project_id ?? resource.filename;
  return {
    key,
    label: resource.project_name || resource.filename,
    filename: resource.filename,
    project_name: resource.project_name,
  };
}

export function serverModToModConfig(entry: GameServerFileEntry): ModConfigMod {
  return {
    key: entry.path,
    label: entry.name.replace(/\.jar$/i, ''),
    filename: entry.name,
  };
}
