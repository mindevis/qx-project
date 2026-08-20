import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Empty,
  Input,
  Modal,
  Radio,
  Spin,
  Tag,
  Typography,
  Upload,
} from 'antd';
import {
  ArrowLeftOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  FolderOpenOutlined,
  ReloadOutlined,
  RightOutlined,
  SearchOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { api, type GameServerFileEntry, type InstanceResource } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { useModal } from '@/hooks/useModal';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { gameServerSyncTargetKey, loadGameServerSyncTargets } from '@/lib/gameServerSyncTargets';
import { formatFileSize } from '@/lib/formatFileSize';
import {
  CLIENT_CONFIG_ROOT,
  CONFIG_MAX_BYTES,
  configFileExtension,
  configRelativePath,
  filterGroupedConfigs,
  groupConfigFilesByMod,
  instanceConfigDestPath,
  joinConfigRoot,
  listConfigPaths,
  sanitizeUploadRelativePath,
  type ModConfigFileEntry,
  type ModConfigMod,
} from '@/lib/modConfigDiscovery';
import { modalMotionProps } from '@/lib/modal';
import './ModConfigsByModPanel.css';

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
  configRoot?: string;
  allowUpload?: boolean;
};

type ConfigTab = {
  key: string;
  label: string;
  mod?: ModConfigMod;
  files: ModConfigFileEntry[];
};

function basename(path: string): string {
  const parts = path.split('/');
  return parts[parts.length - 1] ?? path;
}

function formatSize(size?: number): string {
  return formatFileSize(size) || '—';
}

export function ModConfigsByModPanel({
  mode,
  available,
  mods,
  fileApi,
  configSync,
  configRoot = 'config',
  allowUpload = false,
}: ModConfigsByModPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const modal = useModal();
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
  const [uploading, setUploading] = useState(false);
  const uploadBatchRef = useRef<File[]>([]);
  const uploadFlushScheduled = useRef(false);
  const [serverPickerOpen, setServerPickerOpen] = useState(false);
  const [serverTargets, setServerTargets] = useState<Awaited<ReturnType<typeof loadGameServerSyncTargets>>>([]);
  const [serverTargetsLoading, setServerTargetsLoading] = useState(false);
  const [selectedServerKey, setSelectedServerKey] = useState<string>();
  const userPickedTab = useRef(false);

  const dirty = selectedFile != null && fileContent !== savedContent;
  const otherLabel = t('qxmods.configSync.otherTab');

  const grouped = useMemo(
    () => filterGroupedConfigs(groupConfigFilesByMod(mods, configFiles), searchQuery, otherLabel),
    [configFiles, mods, otherLabel, searchQuery],
  );

  const totalConfigCount = useMemo(
    () => grouped.groups.reduce((sum, group) => sum + group.files.length, 0) + grouped.other.length,
    [grouped],
  );

  const tabItems = useMemo(() => {
    const items: ConfigTab[] = grouped.groups.map((group) => ({
      key: group.mod.key,
      label: group.mod.label,
      mod: group.mod,
      files: group.files,
    }));
    if (grouped.other.length > 0) {
      items.push({
        key: 'other',
        label: otherLabel,
        files: grouped.other,
      });
    }
    return items;
  }, [grouped, otherLabel]);

  const effectiveActiveTab = useMemo(() => {
    if (tabItems.length === 0) return '';
    if (!userPickedTab.current) {
      const preferred =
        tabItems.find((item) => item.key !== 'other' && item.files.length > 0) ??
        tabItems.find((item) => item.files.length > 0) ??
        tabItems[0];
      return preferred.key;
    }
    if (tabItems.some((item) => item.key === activeTab)) {
      return activeTab;
    }
    return (tabItems.find((item) => item.files.length > 0) ?? tabItems[0]).key;
  }, [activeTab, tabItems]);

  const activeItem = tabItems.find((item) => item.key === effectiveActiveTab);
  const activeFiles = activeItem?.files ?? [];

  const loadConfigTree = useCallback(async () => {
    if (!available) {
      setConfigFiles([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const files = await listConfigPaths(async (path) => fileApi.listDir(path), 3, configRoot);
      setConfigFiles(files);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.filesLoadFailed'));
      setConfigFiles([]);
    } finally {
      setLoading(false);
    }
  }, [available, configRoot, fileApi, message, t]);

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
    modal.confirm({
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

  const saveFile = useCallback(async () => {
    if (!selectedFile || fileContent === savedContent) return;
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
  }, [fileApi, fileContent, message, savedContent, selectedFile, t]);

  useEffect(() => {
    if (!selectedFile) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        void saveFile();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [saveFile, selectedFile]);

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
    if (mode !== 'instance') return null;
    if (!configSync?.deviceLinked) return t('qxmods.configSync.syncUnavailableDevice');
    if (!configSync.instanceLoader) return t('qxmods.configSync.syncUnavailableLoader');
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
      await api.writeInstanceFile(configSync.instanceId, instanceConfigDestPath(selectedFile), content);
      setFileContent(content);
      setSavedContent(content);
      message.success(t('qxmods.configSync.success'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.configSync.failed'));
    } finally {
      setSyncing(false);
    }
  };

  const uploadConfigFiles = async (files: File[]) => {
    if (!allowUpload || files.length === 0) return;
    setUploading(true);
    let uploaded = 0;
    let skipped = 0;
    try {
      for (const file of files) {
        const rel = sanitizeUploadRelativePath(file);
        if (!rel || file.size > CONFIG_MAX_BYTES) {
          skipped += 1;
          continue;
        }
        try {
          const content = await file.text();
          if (new TextEncoder().encode(content).length > CONFIG_MAX_BYTES) {
            skipped += 1;
            continue;
          }
          await fileApi.writeFile(joinConfigRoot(configRoot, rel), content);
          uploaded += 1;
        } catch {
          skipped += 1;
        }
      }
      if (uploaded > 0) {
        message.success(t('qxmods.configSync.uploadSuccess', { count: uploaded }));
        await loadConfigTree();
      }
      if (skipped > 0) {
        message.warning(t('qxmods.configSync.uploadSkipped', { count: skipped }));
      }
      if (uploaded === 0 && skipped === 0) {
        message.warning(t('qxmods.configSync.uploadEmpty'));
      }
    } finally {
      setUploading(false);
    }
  };

  const enqueueUploadFiles = (file: File) => {
    uploadBatchRef.current.push(file);
    if (uploadFlushScheduled.current) {
      return false;
    }
    uploadFlushScheduled.current = true;
    queueMicrotask(() => {
      const batch = uploadBatchRef.current;
      uploadBatchRef.current = [];
      uploadFlushScheduled.current = false;
      void uploadConfigFiles(batch);
    });
    return false;
  };

  const selectTab = (key: string) => {
    userPickedTab.current = true;
    setActiveTab(key);
  };

  const clientScope = configRoot === CLIENT_CONFIG_ROOT;
  const title = mode === 'instance'
    ? t('qxmods.configSync.instanceTitle')
    : clientScope
      ? t('qxmods.configSync.clientTitle')
      : t('qxmods.configSync.serverTitle');
  const intro = mode === 'instance'
    ? t('qxmods.configSync.instanceIntro')
    : clientScope
      ? t('qxmods.configSync.clientIntro')
      : t('qxmods.configSync.serverIntro');
  const emptyHintKey =
    mode === 'server'
      ? clientScope
        ? 'qxmods.configSync.emptyHintClient'
        : 'qxmods.configSync.emptyHintServer'
      : 'qxmods.configSync.emptyHint';

  const uploadButtons = allowUpload ? (
    <div className="mod-configs-toolbar-actions">
      <Upload
        multiple
        accept=".toml,.json,.properties,.cfg,.yml,.yaml"
        showUploadList={false}
        disabled={uploading}
        beforeUpload={enqueueUploadFiles}
      >
        <Button icon={<UploadOutlined />} loading={uploading}>
          {t('qxmods.configSync.uploadFiles')}
        </Button>
      </Upload>
      <Upload
        directory
        showUploadList={false}
        disabled={uploading}
        beforeUpload={enqueueUploadFiles}
      >
        <Button icon={<FolderOpenOutlined />} loading={uploading}>
          {t('qxmods.configSync.uploadFolder')}
        </Button>
      </Upload>
    </div>
  ) : null;

  if (!available) {
    return (
      <section className="mod-configs">
        <header className="mod-configs-hero">
          <div className="mod-configs-hero-main">
            <Typography.Title level={3} className="mod-configs-title">
              {title}
            </Typography.Title>
            <Typography.Paragraph type="secondary" className="mod-configs-intro">
              {intro}
            </Typography.Paragraph>
          </div>
        </header>
        <Alert
          type="info"
          showIcon
          message={
            mode === 'instance'
              ? t('launcher.instanceSettingsModsConfigNote')
              : t('servers.gameServersAgentRequired')
          }
        />
      </section>
    );
  }

  const serverPicker = (
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
  );

  if (selectedFile) {
    return (
      <section className="mod-configs">
        <div className="mod-configs-editor">
          <div className="mod-configs-editor-bar">
            <div className="mod-configs-editor-heading">
              <Button type="text" icon={<ArrowLeftOutlined />} onClick={requestCloseEditor}>
                {t('qxmods.configSync.backToList')}
              </Button>
              <Typography.Title level={4} className="mod-configs-editor-name">
                {basename(selectedFile)}
              </Typography.Title>
              <span className="mod-configs-editor-path">{selectedFile}</span>
            </div>
            <div className="mod-configs-editor-actions">
              {dirty ? <Tag color="warning">{t('qxmods.configSync.unsavedChanges')}</Tag> : null}
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
            </div>
          </div>
          {canSyncToServer ? (
            <Typography.Paragraph type="secondary" className="mod-configs-sync-hint">
              {t('qxmods.configSync.syncHintInstance')}
            </Typography.Paragraph>
          ) : canPullFromServer ? (
            <Typography.Paragraph type="secondary" className="mod-configs-sync-hint">
              {t('qxmods.configSync.syncHintServer')}
            </Typography.Paragraph>
          ) : syncUnavailableReason ? (
            <Alert type="info" showIcon message={syncUnavailableReason} />
          ) : null}
          {fileLoading ? (
            <div className="servers-loading">
              <Spin />
            </div>
          ) : (
            <Input.TextArea
              className="mod-configs-textarea"
              value={fileContent}
              rows={18}
              spellCheck={false}
              aria-label={t('qxmods.configSync.editorAria')}
              onChange={(e) => setFileContent(e.target.value)}
            />
          )}
        </div>
        {serverPicker}
      </section>
    );
  }

  return (
    <section className="mod-configs">
      <header className="mod-configs-hero">
        <div className="mod-configs-hero-main">
          <Typography.Title level={3} className="mod-configs-title">
            {title}
          </Typography.Title>
          <Typography.Paragraph type="secondary" className="mod-configs-intro">
            {intro}
          </Typography.Paragraph>
        </div>
        {!loading && totalConfigCount > 0 ? (
          <div className="mod-configs-chips">
            <span className="mod-configs-chip">
              {t('qxmods.configSync.modsChip', { count: tabItems.length })}
            </span>
            <span className="mod-configs-chip">
              {t('qxmods.configSync.filesChip', { count: totalConfigCount })}
            </span>
          </div>
        ) : null}
      </header>

      <div className="mod-configs-toolbar">
        <Input.Search
          allowClear
          className="mod-configs-search"
          prefix={<SearchOutlined aria-hidden />}
          placeholder={t('qxmods.configSync.searchPlaceholder')}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        {uploadButtons}
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void loadConfigTree()}>
          {t('qxmods.configSync.refresh')}
        </Button>
      </div>

      {loading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : totalConfigCount === 0 ? (
        <div className="mod-configs-empty">
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <div>
                <Typography.Text>{t('qxmods.configSync.noConfigs')}</Typography.Text>
                <Typography.Paragraph type="secondary">
                  {t(emptyHintKey)}
                </Typography.Paragraph>
                {allowUpload ? (
                  <Typography.Paragraph type="secondary">
                    {t('qxmods.configSync.uploadHint')}
                  </Typography.Paragraph>
                ) : null}
              </div>
            }
          />
        </div>
      ) : (
        <div className="mod-configs-workspace">
          <aside className="mod-configs-mods">
            <p className="mod-configs-mods-title">{t('qxmods.configSync.modsListTitle')}</p>
            <ul className="mod-configs-mods-list">
              {tabItems.map((item) => {
                const active = item.key === effectiveActiveTab;
                return (
                  <li key={item.key}>
                    <button
                      type="button"
                      className={`mod-configs-mod${active ? ' mod-configs-mod--active' : ''}`}
                      aria-pressed={active}
                      onClick={() => selectTab(item.key)}
                    >
                      <ModCatalogIcon url={item.mod?.icon_url} name={item.label} size={32} />
                      <span className="mod-configs-mod-copy">
                        <span className="mod-configs-mod-name">{item.label}</span>
                        <span className="mod-configs-mod-meta">
                          {t('qxmods.configSync.filesChip', { count: item.files.length })}
                        </span>
                      </span>
                      <span className="mod-configs-mod-count">{item.files.length}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </aside>
          <div className="mod-configs-files">
            <p className="mod-configs-files-title">
              {activeItem
                ? t('qxmods.configSync.filesFor', { name: activeItem.label })
                : t('qxmods.configSync.filesChip', { count: 0 })}
            </p>
            {activeFiles.length === 0 ? (
              <div className="mod-configs-empty">
                <Empty description={t('qxmods.configSync.emptyTabHint')} />
              </div>
            ) : (
              <ul className="mod-configs-file-list">
                {activeFiles.map((row) => {
                  const ext = configFileExtension(row.path);
                  return (
                    <li key={row.path}>
                      <button
                        type="button"
                        className="mod-configs-file"
                        onClick={() => void openFile(row.path)}
                      >
                        <span className="mod-configs-file-ext">{ext || 'cfg'}</span>
                        <span className="mod-configs-file-copy">
                          <span className="mod-configs-file-name">{basename(row.path)}</span>
                          <span className="mod-configs-file-path">{configRelativePath(row.path)}</span>
                        </span>
                        <span className="mod-configs-file-size">{formatSize(row.size)}</span>
                        <RightOutlined aria-hidden />
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </div>
      )}
      {serverPicker}
    </section>
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

export function prettyModFileLabel(filename: string): string {
  const withoutExt = filename.replace(/\.(jar|zip|mrpack)$/i, '');
  const stripped = withoutExt
    .replace(/[-_+]v?\d+(\.\d+)*([+_].*)?$/i, '')
    .replace(/[-_]+$/g, '')
    .trim();
  return stripped || withoutExt;
}

export function serverModToModConfig(entry: GameServerFileEntry): ModConfigMod {
  return {
    key: entry.path,
    label: prettyModFileLabel(entry.name),
    filename: entry.name,
  };
}
