import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Modal,
  Select,
  Segmented,
  Spin,
  Table,
  Typography,
  Upload,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { SearchOutlined, UploadOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  api,
  type GameServerContentKind,
  type GameServerFileEntry,
  type ModCatalogItem,
  type ModCatalogSort,
  type ModCatalogSourceFilter,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  pluginLoaderForServerType,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { cachedListModVersions } from '@/lib/modCatalogCache';
import { isModOnServer } from '@/lib/modSync';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';
import './InstanceResourcesPanel.css';

const { Paragraph, Title } = Typography;
const PAGE_SIZE = 20;

type GameServerContentPanelProps = {
  kind: GameServerContentKind;
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
  supported: boolean;
  serverType: VpsGameServerType;
  mcVersion: string;
};

function formatFileSize(size?: number): string {
  if (size == null || size <= 0) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function projectTypeForKind(kind: GameServerContentKind): ModProjectType {
  switch (kind) {
    case 'plugin':
      return 'plugin';
    case 'datapack':
      return 'datapack';
    default:
      return 'mod';
  }
}

function listInstalled(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
): Promise<{ items: GameServerFileEntry[] }> {
  switch (kind) {
    case 'plugin':
      return api.listVpsGameServerPlugins(vpsId, gameServerId);
    case 'datapack':
      return api.listVpsGameServerDatapacks(vpsId, gameServerId);
    default:
      return Promise.all([
        api.listVpsGameServerMods(vpsId, gameServerId),
        api.listVpsGameServerClientMods(vpsId, gameServerId),
      ]).then(([modsRes, clientRes]) => ({
        items: [...(modsRes.items ?? []), ...(clientRes.items ?? [])],
      }));
  }
}

function modTargetFromPath(path: string): 'mods' | 'client-mods' | undefined {
  const normalized = path.replace(/\\/g, '/').toLowerCase();
  if (normalized.startsWith('client-mods/')) return 'client-mods';
  if (normalized.startsWith('mods/')) return 'mods';
  return undefined;
}

function deleteContent(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
  filename: string,
  modTarget?: 'mods' | 'client-mods',
) {
  switch (kind) {
    case 'plugin':
      return api.deleteVpsGameServerPlugin(vpsId, gameServerId, { filename });
    case 'datapack':
      return api.deleteVpsGameServerDatapack(vpsId, gameServerId, { filename });
    default:
      return api.deleteVpsGameServerMod(vpsId, gameServerId, {
        filename,
        mod_target: modTarget,
      });
  }
}

function syncContent(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
  body: Parameters<typeof api.syncModToGameServer>[2],
) {
  switch (kind) {
    case 'plugin':
      return api.syncPluginToGameServer(vpsId, gameServerId, body);
    case 'datapack':
      return api.syncDatapackToGameServer(vpsId, gameServerId, body);
    default:
      return api.syncModToGameServer(vpsId, gameServerId, body);
  }
}

export function GameServerContentPanel({
  kind,
  vpsId,
  gameServerId,
  agentOnline,
  supported,
  serverType,
  mcVersion,
}: GameServerContentPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const projectType = projectTypeForKind(kind);
  const loader =
    kind === 'plugin'
      ? pluginLoaderForServerType(serverType)
      : kind === 'mod'
        ? serverType
        : undefined;

  const [installed, setInstalled] = useState<GameServerFileEntry[]>([]);
  const [installedLoading, setInstalledLoading] = useState(true);
  const [sourceFilter, setSourceFilter] = useState<ModCatalogSourceFilter>('all');
  const [sort, setSort] = useState<ModCatalogSort>('downloads');
  const [searchInput, setSearchInput] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [catalogItems, setCatalogItems] = useState<ModCatalogItem[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogLoaded, setCatalogLoaded] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [offset, setOffset] = useState(0);
  const [curseforgeEnabled, setCurseforgeEnabled] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailItem, setDetailItem] = useState<ModCatalogItem | null>(null);
  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [installingVersionId, setInstallingVersionId] = useState<string>();
  const [uploading, setUploading] = useState(false);
  const [deletingPath, setDeletingPath] = useState<string>();

  const i18nPrefix = `gameServerDetail.content.${kind}`;

  const loadInstalled = useCallback(async () => {
    if (!agentOnline || !supported) {
      setInstalled([]);
      setInstalledLoading(false);
      return;
    }
    setInstalledLoading(true);
    try {
      const res = await listInstalled(kind, vpsId, gameServerId);
      setInstalled((res.items ?? []).filter((item) => !item.dir));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t(`${i18nPrefix}.loadFailed`));
    } finally {
      setInstalledLoading(false);
    }
  }, [agentOnline, gameServerId, i18nPrefix, kind, message, supported, t, vpsId]);

  useEffect(() => {
    void loadInstalled();
  }, [loadInstalled]);

  const loadCatalog = useCallback(
    async (append = false) => {
      if (!agentOnline || !supported) return;
      setCatalogLoading(true);
      try {
        const nextOffset = append ? offset : 0;
        if (appliedSearch.trim()) {
          const res = await api.searchMods({
            q: appliedSearch.trim(),
            type: projectType,
            loader,
            mc_version: mcVersion,
            limit: PAGE_SIZE,
          });
          setCatalogItems(res.items ?? []);
          setHasMore(false);
          setOffset(0);
          setCurseforgeEnabled(res.curseforge_enabled ?? false);
        } else {
          const res = await api.browseMods({
            type: projectType,
            loader,
            mc_version: mcVersion,
            source: sourceFilter,
            sort,
            limit: PAGE_SIZE,
            offset: nextOffset,
          });
          setCatalogItems((prev) => (append ? [...prev, ...(res.items ?? [])] : res.items ?? []));
          setHasMore(res.has_more ?? false);
          setOffset(nextOffset + (res.items?.length ?? 0));
          setCurseforgeEnabled(res.curseforge_enabled ?? false);
        }
        setCatalogLoaded(true);
      } catch (e) {
        message.error(formatModCatalogError(e, t, `${i18nPrefix}.browseFailed`));
        if (!append) {
          setCatalogItems([]);
          setHasMore(false);
        }
      } finally {
        setCatalogLoading(false);
      }
    },
    [
      agentOnline,
      appliedSearch,
      i18nPrefix,
      loader,
      mcVersion,
      message,
      offset,
      projectType,
      sort,
      sourceFilter,
      supported,
      t,
    ],
  );

  useEffect(() => {
    void loadCatalog(false);
  }, [appliedSearch, sourceFilter, sort, projectType, loader, mcVersion]);

  const openDetail = async (item: ModCatalogItem) => {
    setDetailItem(item);
    setDetailOpen(true);
    setVersions([]);
    setVersionsLoading(true);
    try {
      const items = await cachedListModVersions(item.source as ModSource, item.id, {
        loader,
        mc_version: mcVersion,
      });
      setVersions(items);
    } catch (e) {
      message.error(formatModCatalogError(e, t, `${i18nPrefix}.versionsFailed`));
    } finally {
      setVersionsLoading(false);
    }
  };

  const promptRestart = () => {
    Modal.confirm({
      title: t('gameServerDetail.content.restartTitle'),
      content: t('gameServerDetail.content.restartPrompt'),
      okText: t('gameServerDetail.content.restartConfirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        await restartVpsGameServer(vpsId, gameServerId);
        message.success(t('servers.gameServerRestartStarted'));
      },
    });
  };

  const installVersion = async (item: ModCatalogItem, version: ModVersion) => {
    const file = version.files[0];
    if (!file) {
      message.error(t('gameServerDetail.content.noFile'));
      return;
    }
    setInstallingVersionId(version.id);
    try {
      const res = await syncContent(kind, vpsId, gameServerId, {
        source: item.source,
        project_id: item.id,
        version_id: version.id,
        filename: file.filename,
        download_url: file.url,
        project_name: item.name,
        version_number: version.version_number,
      });
      if (res.status === 'already_installed') {
        message.info(t('gameServerDetail.content.alreadyInstalled'));
      } else {
        message.success(t('gameServerDetail.content.installed'));
        promptRestart();
        void loadInstalled();
      }
      setDetailOpen(false);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.content.installFailed'));
    } finally {
      setInstallingVersionId(undefined);
    }
  };

  const catalogColumns: ColumnsType<ModCatalogItem> = useMemo(
    () => [
      {
        title: t('gameServerDetail.content.catalogName'),
        key: 'name',
        render: (_, row) => (
          <button type="button" className="qxmods-catalog-link" onClick={() => void openDetail(row)}>
            {row.name}
          </button>
        ),
      },
      {
        title: t('gameServerDetail.content.catalogSource'),
        key: 'source',
        render: (_, row) => <ModSourceBadge source={row.source} />,
      },
      {
        title: t('gameServerDetail.content.catalogDownloads'),
        key: 'downloads',
        render: (_, row) => (row.downloads != null ? row.downloads.toLocaleString() : '—'),
      },
    ],
    [t],
  );

  const handleDelete = (row: GameServerFileEntry) => {
    Modal.confirm({
      title: t('gameServerDetail.content.deleteTitle'),
      content: t('gameServerDetail.content.deleteConfirm', { name: row.name }),
      okText: t('gameServerDetail.content.deleteAction'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        setDeletingPath(row.path);
        try {
          await deleteContent(kind, vpsId, gameServerId, row.name, modTargetFromPath(row.path));
          message.success(t('gameServerDetail.content.deleteCompleted'));
          void loadInstalled();
          promptRestart();
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('gameServerDetail.content.deleteFailed'));
          throw e;
        } finally {
          setDeletingPath(undefined);
        }
      },
    });
  };

  const handleUpload = async (file: File) => {
    if (kind !== 'mod') return false;
    setUploading(true);
    try {
      await api.uploadGameServerMod(vpsId, gameServerId, file);
      message.success(t('gameServerDetail.content.uploadCompleted'));
      void loadInstalled();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.content.uploadFailed'));
    } finally {
      setUploading(false);
    }
    return false;
  };

  if (!supported) {
    return <Paragraph type="secondary">{t(`${i18nPrefix}.notSupported`)}</Paragraph>;
  }

  if (!agentOnline) {
    return <Paragraph type="secondary">{t('servers.gameServersAgentRequired')}</Paragraph>;
  }

  const showCurseforgeUnavailable =
    sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled && !appliedSearch.trim();

  return (
    <div className="game-server-content-panel">
      <div className="game-server-content-installed-header">
        <Title level={5}>{t(`${i18nPrefix}.installedTitle`)}</Title>
        {kind === 'mod' ? (
          <Upload
            accept=".jar,.zip,.mrpack"
            showUploadList={false}
            disabled={uploading}
            beforeUpload={(file) => {
              void handleUpload(file);
              return false;
            }}
          >
            <Button icon={<UploadOutlined />} loading={uploading}>
              {t('gameServerDetail.content.upload')}
            </Button>
          </Upload>
        ) : null}
      </div>
      {installedLoading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : installed.length === 0 ? (
        <Empty description={t(`${i18nPrefix}.empty`)} />
      ) : (
        <Table
          className="game-server-mods-table"
          rowKey="path"
          size="small"
          pagination={false}
          dataSource={installed}
          columns={[
            { title: t('gameServerDetail.fileName'), dataIndex: 'name', key: 'name' },
            ...(kind === 'mod'
              ? [
                  {
                    title: t('gameServerDetail.content.folder'),
                    key: 'folder',
                    render: (_: unknown, row: GameServerFileEntry) => {
                      const target = modTargetFromPath(row.path);
                      return target === 'client-mods'
                        ? t('gameServerDetail.content.clientModsFolder')
                        : t('gameServerDetail.content.modsFolder');
                    },
                  },
                ]
              : []),
            {
              title: t('gameServerDetail.fileSize'),
              key: 'size',
              render: (_, row) => formatFileSize(row.size),
            },
            {
              title: '',
              key: 'actions',
              width: 56,
              render: (_, row) => (
                <Button
                  type="text"
                  danger
                  size="small"
                  icon={<DeleteOutlined />}
                  loading={deletingPath === row.path}
                  aria-label={t('gameServerDetail.content.deleteAction')}
                  onClick={() => handleDelete(row)}
                />
              ),
            },
          ]}
        />
      )}

      <Title level={5} className="game-server-content-browse-title">
        {t(`${i18nPrefix}.browseTitle`)}
      </Title>
      <div className="qxmods-catalog-toolbar">
        <Input
          allowClear
          prefix={<SearchOutlined aria-hidden />}
          placeholder={t(`${i18nPrefix}.searchPlaceholder`)}
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          onPressEnter={() => setAppliedSearch(searchInput.trim())}
        />
        <Button type="primary" onClick={() => setAppliedSearch(searchInput.trim())}>
          {t('gameServerDetail.content.search')}
        </Button>
        <Segmented
          value={sourceFilter}
          onChange={(value) => setSourceFilter(value as ModCatalogSourceFilter)}
          options={[
            { value: 'all', label: t('gameServerDetail.content.sourceAll') },
            { value: 'curseforge', label: 'CurseForge' },
            { value: 'modrinth', label: 'Modrinth' },
          ]}
        />
        <Select
          value={sort}
          onChange={(value) => setSort(value as ModCatalogSort)}
          options={[
            { value: 'downloads', label: t('gameServerDetail.content.sortDownloads') },
            { value: 'newest', label: t('gameServerDetail.content.sortNewest') },
            { value: 'updated', label: t('gameServerDetail.content.sortUpdated') },
          ]}
        />
      </div>
      {showCurseforgeUnavailable ? (
        <Typography.Text type="secondary">
          {t('gameServerDetail.content.curseforgeUnavailable')}
        </Typography.Text>
      ) : null}
      {catalogLoading && catalogItems.length === 0 ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : catalogItems.length === 0 ? (
        <Empty description={t(`${i18nPrefix}.browseEmpty`)} />
      ) : (
        <>
          <Table
            className="qxmods-catalog-table"
            rowKey={(row) => `${row.source}:${row.id}`}
            size="small"
            pagination={false}
            dataSource={catalogItems}
            columns={catalogColumns}
          />
          {hasMore && !appliedSearch.trim() ? (
            <Button loading={catalogLoading} onClick={() => void loadCatalog(true)}>
              {t('gameServerDetail.content.loadMore')}
            </Button>
          ) : null}
        </>
      )}

      <Modal
        {...modalMotionProps}
        title={detailItem?.name ?? t(`${i18nPrefix}.detailTitle`)}
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={640}
      >
        {detailItem ? (
          <>
            <Paragraph type="secondary">{detailItem.summary}</Paragraph>
            {versionsLoading ? (
              <Spin />
            ) : versions.length === 0 ? (
              <Empty description={t(`${i18nPrefix}.versionsEmpty`)} />
            ) : (
              <Table
                size="small"
                rowKey="id"
                pagination={false}
                dataSource={versions}
                columns={[
                  {
                    title: t('gameServerDetail.content.version'),
                    dataIndex: 'version_number',
                    key: 'version_number',
                  },
                  {
                    title: '',
                    key: 'action',
                    render: (_, version) => {
                      const onServer = isModOnServer(installed, version);
                      return (
                        <Button
                          type="primary"
                          size="small"
                          disabled={onServer}
                          loading={installingVersionId === version.id}
                          onClick={() => void installVersion(detailItem, version)}
                        >
                          {onServer
                            ? t('gameServerDetail.content.alreadyInstalled')
                            : t('gameServerDetail.content.install')}
                        </Button>
                      );
                    },
                  },
                ]}
              />
            )}
          </>
        ) : null}
      </Modal>
    </div>
  );
}
