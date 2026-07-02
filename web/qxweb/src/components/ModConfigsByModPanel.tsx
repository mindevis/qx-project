import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Badge,
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
import {
  ArrowLeftOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  FileOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { api, type GameServerFileEntry, type InstanceResource } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { gameServerSyncTargetKey, loadGameServerSyncTargets } from '@/lib/gameServerSyncTargets';
import { formatFileSize } from '@/lib/formatFileSize';
import {
  configRelativePath,
  filterConfigFileEntries,
  groupConfigFilesByMod,
  listConfigPaths,
  type ModConfigFileEntry,
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

function formatSize(size?: number): string {
  const formatted = formatFileSize(size);
  return formatted || '—';
}

function renderTabLabel(label: string, count: number, iconUrl?: string) {
  return (
    <span className="mod-configs-tab-label">
      {iconUrl ? <ModCatalogIcon url={iconUrl} name={label} size={18} /> : null}
      <span>{label}</span>
      {count > 0 ? (
        <Badge count={count} size="small" color="default" className="mod-configs-tab-badge" />
      ) : null}
    </span>
  );
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
  const [configFiles, setConfigFiles] = useState<ModConfigFileEntry[]>([]);
  const [activeTab, setActiveTab] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [savedContent, setSavedContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [serverPickerOpen, setServerPickerOpen] = useState(false);
  const [serverTargets, setServerTargets] = useState<Awaited<ReturnType<typeof loadGameServerSyncTargets>>>([]);
  const [serverTargetsLoading, setServerTargetsLoading] = useState(false);
  const [selectedServerKey, setSelectedServerKey] = useState<string>();
  const userPickedTab = useRef(false);

  const dirty = selectedFile != null && fileContent !== savedContent;

  const grouped = useMemo(
    () => groupConfigFilesByMod(mods, configFiles),
    [configFiles, mods],
  );

  const totalConfigCount = useMemo(
    () => grouped.groups.reduce((sum, group) => sum + group.files.length, 0) + grouped.other.length,
    [grouped],
  );

  const tabItems = useMemo(() => {
    const items: Array<{
      key: string;
      label: string;
      mod?: ModConfigMod;
      files: ModConfigFileEntry[];
    }> = grouped.groups.map((group) => ({
      key: group.mod.key,
      label: group.mod.label,
      mod: group.mod,
      files: group.files,
    }));
    items.push({
      key: 'other',
      label: t('qxmods.configSync.otherTab'),
      files: grouped.other,
    });
    return items;
  }, [grouped, t]);

  const effectiveActiveTab = useMemo(() => {
    if (tabItems.length === 0) return '';
    if (!userPickedTab.current) {
      const preferred =
        tabItems.find((item) => item.key !== 'other' && item.files.length > 0) ??
        tabItems.find((item) => item.files.length > 0) ??
        tabItems[0];
      return preferred.key;
    }
    let resolved = tabItems.some((item) => item.key === activeTab) ? activeTab : '';
    if (!resolved) {
      const preferred =
        tabItems.find((item) => item.key !== 'other' && item.files.length > 0) ?? tabItems[0];
      resolved = preferred.key;
    }
    const current = tabItems.find((item) => item.key === resolved);
    if (current && current.files.length === 0) {
      const fallback = tabItems.find((item) => item.files.length > 0);
      if (fallback) resolved = fallback.key;
    }
    return resolved;
  }, [activeTab, tabItems]);

  const activeFiles = useMemo(
    () => tabItems.find((item) => item.key === effectiveActiveTab)?.files ?? [],
    [effectiveActiveTab, tabItems],
  );

  const filteredFiles = useMemo(
    () => filterConfigFileEntries(activeFiles, searchQuery),
    [activeFiles, searchQuery],
  );

  const loadConfigTree = useCallback(async () => {
    if (!available) {
      setConfigFiles([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const files = await listConfigPaths(async (path) => {
        const res = await fileApi.listDir(path);
        return res;
      });
      setConfigFiles(files);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.filesLoadFailed'));
      setConfigFiles([]);
    } finally {
      setLoading(false);
    }
  }, [available, fileApi, message, t]);

  useEffect(() => {
    void loadConfigTree();
  }, [loadConfigTree]);

  const closeEditor = () => {
    setSelectedFile(null);
    setFileContent('');
    setSavedContent('');
  };

  const requestCloseEditor = () => {
    if (!dirty) {
      closeEditor();
      return;
    }
    Modal.confirm({
      title: t('qxmods.configSync.discardChangesTitle'),
      content: t('qxmods.configSync.discardChangesBody'),
      okText: t('qxmods.configSync.discardChangesConfirm'),
      cancelText: t('common.cancel'),
      onOk: closeEditor,
      ...modalMotionProps,
    });
  };

  const openFile = async (path: string) => {
    setSelectedFile(path);
    setFileLoading(true);
    try {
      const content = await fileApi.readFile(path);
      setFileContent(content);
      setSavedContent(content);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.fileReadFailed'));
      setSelectedFile(null);
      setSavedContent('');
    } finally {
      setFileLoading(false);
    }
  };

  const saveFile = async () => {
    if (!selectedFile || !dirty) return;
    setSaving(true);
    try {
      await fileApi.writeFile(selectedFile, fileContent);
      setSavedContent(fileContent);
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

  const syncUnavailableReason = useMemo(() => {
    if (mode === 'instance') {
      if (!configSync?.deviceLinked) return t('qxmods.configSync.syncUnavailableDevice');
      if (!configSync.instanceLoader) return t('qxmods.configSync.syncUnavailableLoader');
      return null;
    }
    if (!configSync?.deviceLinked) return t('qxmods.configSync.syncUnavailableDevice');
    if (!configSync.instanceId) return t('qxmods.configSync.syncUnavailableBinding');
    if (!configSync.agentOnline) return t('servers.gameServersAgentRequired');
    return null;
  }, [configSync, mode, t]);

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
      const content = dirty ? fileContent : await fileApi.readFile(selectedFile);
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
      const content = dirty ? fileContent : await fileApi.readFile(selectedFile);
      await api.writeInstanceFile(configSync.instanceId, selectedFile, content);
      setFileContent(content);
      setSavedContent(content);
      message.success(t('qxmods.configSync.success'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.configSync.failed'));
    } finally {
      setSyncing(false);
    }
  };

  if (!available) {
    return (
      <Alert
        type="info"
        showIcon
        message={
          mode === 'instance'
            ? t('launcher.instanceSettingsModsConfigNote')
            : t('servers.gameServersAgentRequired')
        }
      />
    );
  }

  if (selectedFile) {
    return (
      <div className="game-server-files-editor mod-configs-editor">
        <Space className="game-server-files-toolbar" wrap>
          <Button icon={<ArrowLeftOutlined />} onClick={requestCloseEditor}>
            {t('gameServerDetail.backToFiles')}
          </Button>
          <Typography.Text code className="mod-configs-editor-path">
            {selectedFile}
          </Typography.Text>
          {dirty ? (
            <Typography.Text type="warning">{t('qxmods.configSync.unsavedChanges')}</Typography.Text>
          ) : null}
          <Button type="primary" loading={saving} disabled={!dirty} onClick={() => void saveFile()}>
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
        {canSyncToServer ? (
          <Typography.Paragraph type="secondary" className="mod-configs-sync-hint">
            {t('qxmods.configSync.syncHintInstance')}
          </Typography.Paragraph>
        ) : canPullFromServer ? (
          <Typography.Paragraph type="secondary" className="mod-configs-sync-hint">
            {t('qxmods.configSync.syncHintServer')}
          </Typography.Paragraph>
        ) : syncUnavailableReason ? (
          <Alert type="info" showIcon className="mod-configs-sync-hint" message={syncUnavailableReason} />
        ) : null}
        {fileLoading ? (
          <div className="servers-loading">
            <Spin />
          </div>
        ) : (
          <Input.TextArea
            className="game-server-files-textarea mod-configs-textarea"
            value={fileContent}
            rows={18}
            spellCheck={false}
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
    <div className="game-server-files mod-configs-panel">
      <div className="mod-configs-toolbar">
        <Input
          allowClear
          className="mod-configs-toolbar-search"
          prefix={<SearchOutlined aria-hidden />}
          placeholder={t('qxmods.configSync.searchPlaceholder')}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void loadConfigTree()}>
          {t('qxmods.configSync.refresh')}
        </Button>
        {!loading && totalConfigCount > 0 ? (
          <Typography.Text type="secondary">
            {t('qxmods.configSync.summary', { count: totalConfigCount })}
          </Typography.Text>
        ) : null}
      </div>
      {mode === 'server' && syncUnavailableReason ? (
        <Alert type="info" showIcon className="mod-configs-binding-hint" message={syncUnavailableReason} />
      ) : null}
      {loading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : totalConfigCount === 0 ? (
        <Empty description={t('qxmods.configSync.noConfigs')}>
          <Typography.Paragraph type="secondary">{t('qxmods.configSync.emptyHint')}</Typography.Paragraph>
        </Empty>
      ) : (
        <>
          <Tabs
            activeKey={effectiveActiveTab}
            onChange={(key) => {
              userPickedTab.current = true;
              setActiveTab(key);
            }}
            tabBarGutter={8}
            className="mod-configs-tabs"
            items={tabItems.map((item) => ({
              key: item.key,
              label: renderTabLabel(item.label, item.files.length, item.mod?.icon_url),
            }))}
          />
          {activeFiles.length === 0 ? (
            <Empty description={t('qxmods.configSync.noConfigs')}>
              <Typography.Paragraph type="secondary">{t('qxmods.configSync.emptyTabHint')}</Typography.Paragraph>
            </Empty>
          ) : filteredFiles.length === 0 ? (
            <Empty description={t('qxmods.configSync.noSearchResults')} />
          ) : (
            <Table
              className="game-server-files-table mod-configs-table"
              rowKey="path"
              size="small"
              pagination={false}
              dataSource={filteredFiles}
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
                      <span>{basename(row.path)}</span>
                    </Space>
                  ),
                },
                {
                  title: t('qxmods.configSync.filePath'),
                  key: 'path',
                  responsive: ['md'],
                  render: (_, row) => (
                    <Typography.Text type="secondary" className="mod-configs-path">
                      {configRelativePath(row.path)}
                    </Typography.Text>
                  ),
                },
                {
                  title: t('gameServerDetail.fileSize'),
                  key: 'size',
                  width: 96,
                  render: (_, row) => formatSize(row.size),
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
    icon_url: resource.icon_url,
  };
}

export function serverModToModConfig(entry: GameServerFileEntry): ModConfigMod {
  return {
    key: entry.path,
    label: entry.name.replace(/\.jar$/i, ''),
    filename: entry.name,
  };
}
