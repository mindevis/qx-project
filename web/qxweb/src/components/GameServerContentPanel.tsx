import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Empty,
  Input,
  Modal,
  Select,
  Segmented,
  Spin,
  Switch,
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
  type InstanceResource,
  type ModCatalogItem,
  type ModCatalogSort,
  type ModCatalogSourceFilter,
  type ModProjectType,
  type ModSource,
} from '@/api/client';
import { GameServerCatalogProvider } from '@/components/GameServerCatalogProvider';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { ModCatalogInstallControls } from '@/components/ModCatalogInstallControls';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  pluginLoaderForServerType,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { formatCompactCount } from '@/lib/formatCompactCount';
import { isCatalogItemOnServer, needsServerRestartAfterSync } from '@/lib/modSync';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';
import './InstanceResourcesPanel.css';

const { Paragraph, Text, Title } = Typography;
const PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 400;

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
  const [installedResources, setInstalledResources] = useState<InstanceResource[]>([]);
  const [installedLoading, setInstalledLoading] = useState(true);
  const [installedSearchInput, setInstalledSearchInput] = useState('');
  const [appliedInstalledSearch, setAppliedInstalledSearch] = useState('');
  const [sourceFilter, setSourceFilter] = useState<ModCatalogSourceFilter>('all');
  const [sort, setSort] = useState<ModCatalogSort>('downloads');
  const [searchInput, setSearchInput] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [catalogItems, setCatalogItems] = useState<ModCatalogItem[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [catalogLoaded, setCatalogLoaded] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [offset, setOffset] = useState(0);
  const [curseforgeEnabled, setCurseforgeEnabled] = useState(false);
  const [section, setSection] = useState<'installed' | 'catalog'>('installed');
  const [showInstalledOnly, setShowInstalledOnly] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailItem, setDetailItem] = useState<ModCatalogItem | null>(null);
  const [uploading, setUploading] = useState(false);
  const [deletingPath, setDeletingPath] = useState<string>();
  const [modTarget, setModTarget] = useState<'mods' | 'client-mods'>('mods');

  const i18nPrefix = `gameServerDetail.content.${kind}`;

  const loadInstalled = useCallback(async () => {
    if (!agentOnline || !supported) {
      setInstalled([]);
      setInstalledResources([]);
      setInstalledLoading(false);
      return;
    }
    setInstalledLoading(true);
    try {
      const [res, resourcesRes] = await Promise.all([
        listInstalled(kind, vpsId, gameServerId),
        api.listGameServerResources(vpsId, gameServerId, { kind: projectType }).catch(() => ({ items: [] })),
      ]);
      setInstalled((res.items ?? []).filter((item) => !item.dir));
      setInstalledResources(resourcesRes.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t(`${i18nPrefix}.loadFailed`));
    } finally {
      setInstalledLoading(false);
    }
  }, [agentOnline, gameServerId, i18nPrefix, kind, message, projectType, supported, t, vpsId]);

  useEffect(() => {
    void loadInstalled();
  }, [loadInstalled]);

  useEffect(() => {
    const trimmed = searchInput.trim();
    if (trimmed === appliedSearch) {
      return;
    }
    const timer = window.setTimeout(() => {
      setAppliedSearch(trimmed);
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [appliedSearch, searchInput]);

  useEffect(() => {
    if (!agentOnline || !supported) {
      setCatalogItems([]);
      setCatalogLoaded(false);
      return;
    }
    let cancelled = false;
    setCatalogLoaded(false);
    void (async () => {
      setCatalogLoading(true);
      setLoadingMore(false);
      try {
        if (appliedSearch.trim()) {
          const res = await api.searchMods({
            q: appliedSearch.trim(),
            type: projectType,
            loader,
            mc_version: mcVersion,
            source: sourceFilter,
            limit: PAGE_SIZE,
          });
          if (cancelled) return;
          setCatalogItems(res.items ?? []);
          setHasMore(false);
          setOffset(0);
          setCurseforgeEnabled(res.curseforge_enabled ?? false);
          return;
        }
        const res = await api.browseMods({
          type: projectType,
          loader,
          mc_version: mcVersion,
          source: sourceFilter,
          sort,
          limit: PAGE_SIZE,
          offset: 0,
        });
        if (cancelled) return;
        const nextItems = res.items ?? [];
        setCatalogItems(nextItems);
        setHasMore(res.has_more ?? false);
        setOffset(nextItems.length);
        setCurseforgeEnabled(res.curseforge_enabled ?? false);
      } catch (e) {
        if (cancelled) return;
        message.error(formatModCatalogError(e, t, `${i18nPrefix}.browseFailed`));
        setCatalogItems([]);
        setHasMore(false);
        setOffset(0);
      } finally {
        if (!cancelled) {
          setCatalogLoading(false);
          setCatalogLoaded(true);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [agentOnline, appliedSearch, i18nPrefix, loader, mcVersion, message, projectType, sort, sourceFilter, supported, t]);

  const loadMore = async () => {
    if (appliedSearch.trim() || !agentOnline || !supported) return;
    setLoadingMore(true);
    try {
      const res = await api.browseMods({
        type: projectType,
        loader,
        mc_version: mcVersion,
        source: sourceFilter,
        sort,
        limit: PAGE_SIZE,
        offset,
      });
      const nextItems = res.items ?? [];
      setCatalogItems((prev) => [...prev, ...nextItems]);
      setHasMore(res.has_more ?? false);
      setOffset((prev) => prev + nextItems.length);
      setCurseforgeEnabled(res.curseforge_enabled ?? false);
    } catch (e) {
      message.error(formatModCatalogError(e, t, `${i18nPrefix}.browseFailed`));
    } finally {
      setLoadingMore(false);
    }
  };

  const isSearchMode = appliedSearch.trim().length > 0;

  const installedForTarget = useMemo(() => {
    if (kind !== 'mod') return installed;
    return installed.filter((item) => (modTargetFromPath(item.path) ?? 'mods') === modTarget);
  }, [installed, kind, modTarget]);

  const resourcesForTarget = useMemo(() => {
    if (kind !== 'mod') return installedResources;
    const side = modTarget === 'client-mods' ? 'client' : 'server';
    return installedResources.filter((item) => (item.side_override || 'server') === side);
  }, [installedResources, kind, modTarget]);

  const isItemOnServer = useCallback(
    (item: ModCatalogItem) =>
      resourcesForTarget.some(
        (resource) => resource.source === item.source && resource.project_id === item.id,
      ) || isCatalogItemOnServer(item, installedForTarget),
    [installedForTarget, resourcesForTarget],
  );

  const installedProjectIds = useMemo(() => {
    const keys = new Set<string>();
    for (const resource of resourcesForTarget) {
      if (resource.project_id) {
        keys.add(`${resource.source}:${resource.project_id}`);
      }
    }
    for (const item of catalogItems) {
      if (isItemOnServer(item)) {
        keys.add(`${item.source}:${item.id}`);
      }
    }
    return keys;
  }, [catalogItems, isItemOnServer, resourcesForTarget]);

  const visibleCatalogItems = useMemo(() => {
    if (showInstalledOnly) {
      return catalogItems.filter((item) => isItemOnServer(item));
    }
    return catalogItems.filter((item) => !isItemOnServer(item));
  }, [catalogItems, isItemOnServer, showInstalledOnly]);

  const openDetail = useCallback((item: ModCatalogItem) => {
    setDetailItem(item);
    setDetailOpen(true);
  }, []);

  const filteredInstalled = useMemo(() => {
    const query = appliedInstalledSearch.trim().toLowerCase();
    if (!query) return installedForTarget;
    return installedForTarget.filter((item) => {
      const name = item.name.toLowerCase();
      const path = item.path.toLowerCase();
      return name.includes(query) || path.includes(query);
    });
  }, [appliedInstalledSearch, installedForTarget]);

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

  const catalogColumns: ColumnsType<ModCatalogItem> = useMemo(
    () => [
      {
        title: '',
        key: 'icon',
        width: 56,
        render: (_, row) => (
          <ModCatalogIcon url={row.icon_url} name={row.name} size={44} className="qxmods-catalog-table-icon" />
        ),
      },
      {
        title: t('gameServerDetail.content.catalogName'),
        key: 'name',
        width: 220,
        ellipsis: true,
        render: (_, row) => (
          <div className="qxmods-catalog-name-cell">
            <button type="button" className="qxmods-catalog-link" onClick={() => void openDetail(row)}>
              {row.name}
            </button>
            <div className="qxmods-catalog-name-meta">
              <ModSourceBadge source={row.source} />
              {row.author ? (
                <Text type="secondary" className="qxmods-catalog-author">
                  {row.author}
                </Text>
              ) : null}
            </div>
          </div>
        ),
      },
      {
        title: t('qxmods.catalog.summary'),
        dataIndex: 'summary',
        key: 'summary',
        ellipsis: true,
        responsive: ['md'],
      },
      {
        title: t('gameServerDetail.content.catalogDownloads'),
        key: 'downloads',
        width: 96,
        render: (_, row) => (row.downloads != null ? formatCompactCount(row.downloads) : '—'),
      },
      {
        title: t('gameServerDetail.content.install'),
        key: 'install',
        width: 260,
        className: 'qxmods-catalog-install-cell',
        render: (_, row) => (
          <ModCatalogInstallControls
            source={row.source as ModSource}
            projectId={row.id}
            projectName={row.name}
            projectType={row.project_type ?? projectType}
            iconUrl={row.icon_url}
            downloads={row.downloads}
            clientSide={row.client_side}
            serverSide={row.server_side}
            loader={loader}
            mcVersion={mcVersion}
            installedProjectIds={installedProjectIds}
            layout="inline"
            eagerVersions={false}
            selectClassName="qxmods-install-version-select--table"
            onInstalled={() => {
              void loadInstalled();
              if (needsServerRestartAfterSync(modTarget)) {
                promptRestart();
              }
            }}
            onUninstalled={() => void loadInstalled()}
          />
        ),
      },
    ],
    [installedProjectIds, loadInstalled, loader, mcVersion, modTarget, openDetail, projectType, t],
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
          if (needsServerRestartAfterSync(modTargetFromPath(row.path))) {
            promptRestart();
          }
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
      await api.uploadGameServerMod(vpsId, gameServerId, file, modTarget);
      message.success(t('gameServerDetail.content.uploadCompleted'));
      void loadInstalled();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.content.uploadFailed'));
    } finally {
      setUploading(false);
    }
    return false;
  };

  const sourceOptions = useMemo(
    () => [
      { value: 'all', label: t('qxmods.filters.sourceAll') },
      { value: 'modrinth', label: t('qxmods.source.modrinth') },
      { value: 'curseforge', label: t('qxmods.source.curseforge') },
    ],
    [t],
  );

  const sortOptions = useMemo(
    () => [
      { value: 'downloads', label: t('qxmods.filters.sortDownloads') },
      { value: 'newest', label: t('qxmods.filters.sortNewest') },
      { value: 'updated', label: t('qxmods.filters.sortUpdated') },
      { value: 'relevance', label: t('qxmods.filters.sortRelevance') },
    ],
    [t],
  );

  if (!supported) {
    return <Paragraph type="secondary">{t(`${i18nPrefix}.notSupported`)}</Paragraph>;
  }

  if (!agentOnline) {
    return <Paragraph type="secondary">{t('servers.gameServersAgentRequired')}</Paragraph>;
  }

  const showCurseforgeUnavailable =
    sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled;

  return (
    <GameServerCatalogProvider
      kind={kind}
      vpsId={vpsId}
      gameServerId={gameServerId}
      serverType={serverType}
      mcVersion={mcVersion}
      modTarget={kind === 'mod' ? modTarget : undefined}
    >
    <div className="game-server-content-panel">
      <Segmented
        className="game-server-content-sections"
        value={section}
        onChange={(value) => setSection(value as 'installed' | 'catalog')}
        options={[
          { value: 'installed', label: t('gameServerDetail.content.tabInstalled') },
          { value: 'catalog', label: t('gameServerDetail.content.tabCatalog') },
        ]}
      />
      {section === 'installed' ? (
      <>
      <div className="game-server-content-installed-header">
        <div className="game-server-content-installed-header-main">
          <Title level={5}>{t(`${i18nPrefix}.installedTitle`)}</Title>
          <div className="game-server-content-installed-search">
            <Input
              allowClear
              prefix={<SearchOutlined aria-hidden />}
              placeholder={t('qxmods.searchFilterPlaceholder')}
              value={installedSearchInput}
              onChange={(e) => {
                const value = e.target.value;
                setInstalledSearchInput(value);
                if (!value.trim() && appliedInstalledSearch) {
                  setAppliedInstalledSearch('');
                }
              }}
              onPressEnter={() => setAppliedInstalledSearch(installedSearchInput.trim())}
              onClear={() => {
                setInstalledSearchInput('');
                setAppliedInstalledSearch('');
              }}
            />
            <Button
              type="primary"
              onClick={() => setAppliedInstalledSearch(installedSearchInput.trim())}
              disabled={!installedSearchInput.trim() && !appliedInstalledSearch}
            >
              {appliedInstalledSearch ? t('qxmods.applySearch') : t('qxmods.search')}
            </Button>
            {appliedInstalledSearch ? (
              <Button type="link" onClick={() => {
                setInstalledSearchInput('');
                setAppliedInstalledSearch('');
              }}>
                {t('qxmods.clearSearch')}
              </Button>
            ) : null}
          </div>
        </div>
        {kind === 'mod' ? (
          <div className="game-server-content-upload-wrapper">
            <Segmented
              value={modTarget}
              aria-label={t('gameServerDetail.content.folder')}
              onChange={(value) => setModTarget(value as 'mods' | 'client-mods')}
              options={[
                { value: 'mods', label: t('gameServerDetail.content.modsFolder') },
                { value: 'client-mods', label: t('gameServerDetail.content.clientModsFolder') },
              ]}
            />
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
          </div>
        ) : null}
      </div>
      {installedLoading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : installedForTarget.length === 0 ? (
        <Empty description={t(`${i18nPrefix}.empty`)} />
      ) : filteredInstalled.length === 0 ? (
        <Empty description={t('qxmods.empty')} />
      ) : (
        <Table
          className="game-server-mods-table"
          rowKey="path"
          size="small"
          pagination={false}
          dataSource={filteredInstalled}
          columns={[
            { title: t('gameServerDetail.fileName'), dataIndex: 'name', key: 'name' },
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
      </>
      ) : (
      <>
      <div className="qxmods-filters">
        <div className="qxmods-filters-row">
          <label className="qxmods-filter-field">
            <Text type="secondary" className="qxmods-filter-label">
              {t('qxmods.filters.source')}
            </Text>
            <Select
              value={sourceFilter}
              options={sourceOptions}
              onChange={(value) => setSourceFilter(value as ModCatalogSourceFilter)}
              className="qxmods-filter-select"
            />
          </label>
          <label className="qxmods-filter-field">
            <Text type="secondary" className="qxmods-filter-label">
              {t('qxmods.filters.sort')}
            </Text>
            <Select
              value={sort}
              options={sortOptions}
              disabled={isSearchMode}
              onChange={(value) => setSort(value as ModCatalogSort)}
              className="qxmods-filter-select"
            />
          </label>
          {kind === 'mod' ? (
            <label className="qxmods-filter-field">
              <Text type="secondary" className="qxmods-filter-label">
                {t('gameServerDetail.content.folder')}
              </Text>
              <Segmented
                value={modTarget}
                aria-label={t('gameServerDetail.content.folder')}
                onChange={(value) => setModTarget(value as 'mods' | 'client-mods')}
                options={[
                  { value: 'mods', label: t('gameServerDetail.content.modsFolder') },
                  { value: 'client-mods', label: t('gameServerDetail.content.clientModsFolder') },
                ]}
              />
            </label>
          ) : null}
          <label className="qxmods-filter-field qxmods-filter-field--switch">
            <Text type="secondary" className="qxmods-filter-label">
              {t('qxmods.filters.installedOnly')}
            </Text>
            <Switch
              checked={showInstalledOnly}
              onChange={setShowInstalledOnly}
              aria-label={t('qxmods.filters.installedOnly')}
            />
          </label>
        </div>
        <div className="qxmods-search-filter">
          <Input
            allowClear
            prefix={<SearchOutlined aria-hidden />}
            placeholder={t(`${i18nPrefix}.searchPlaceholder`)}
            value={searchInput}
            onChange={(e) => {
              const value = e.target.value;
              setSearchInput(value);
              if (!value.trim() && appliedSearch) {
                setAppliedSearch('');
              }
            }}
            onPressEnter={() => setAppliedSearch(searchInput.trim())}
            onClear={() => {
              setSearchInput('');
              setAppliedSearch('');
            }}
          />
          <Button
            onClick={() => setAppliedSearch(searchInput.trim())}
            disabled={!searchInput.trim() && !appliedSearch}
          >
            {appliedSearch ? t('qxmods.applySearch') : t('qxmods.search')}
          </Button>
          {appliedSearch ? (
            <Button
              type="link"
              onClick={() => {
                setSearchInput('');
                setAppliedSearch('');
              }}
            >
              {t('qxmods.clearSearch')}
            </Button>
          ) : null}
        </div>
      </div>
      <Paragraph type="secondary" className="qxmods-filter-context">
        {t('qxmods.filterContext', { mcVersion, loader: loader ?? serverType })}
      </Paragraph>
      {showCurseforgeUnavailable ? (
        <Alert type="warning" showIcon title={t('qxmods.curseforgeDisabled')} />
      ) : catalogLoading && catalogItems.length === 0 && !showInstalledOnly ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : (
        <>
          <Table
            className="qxmods-catalog-table qxmods-catalog-table--install"
            rowKey={(row) => `${row.source}:${row.id}`}
            size="small"
            pagination={false}
            loading={catalogLoading && !showInstalledOnly}
            dataSource={visibleCatalogItems}
            columns={catalogColumns}
            scroll={{ x: 860 }}
            locale={{
              emptyText: showInstalledOnly
                ? t('qxmods.installed.empty')
                : isSearchMode
                  ? t('qxmods.empty')
                  : t(`${i18nPrefix}.browseEmpty`),
            }}
          />
          {!showInstalledOnly && !isSearchMode && hasMore ? (
            <div className="qxmods-load-more">
              <Button loading={loadingMore} onClick={() => void loadMore()}>
                {t('gameServerDetail.content.loadMore')}
              </Button>
            </div>
          ) : null}
        </>
      )}
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
            <ModCatalogInstallControls
              source={detailItem.source as ModSource}
              projectId={detailItem.id}
              projectName={detailItem.name}
              projectType={detailItem.project_type ?? projectType}
              iconUrl={detailItem.icon_url}
              downloads={detailItem.downloads}
              clientSide={detailItem.client_side}
              serverSide={detailItem.server_side}
              loader={loader}
              mcVersion={mcVersion}
              installedProjectIds={installedProjectIds}
              layout="stacked"
              eagerVersions
              onInstalled={() => {
                setDetailOpen(false);
                void loadInstalled();
                if (needsServerRestartAfterSync(modTarget)) {
                  promptRestart();
                }
              }}
              onUninstalled={() => void loadInstalled()}
            />
          </>
        ) : null}
      </Modal>
    </div>
    </GameServerCatalogProvider>
  );
}
