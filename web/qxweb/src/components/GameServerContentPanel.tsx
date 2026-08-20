import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Empty,
  Input,
  Modal,
  Popconfirm,
  Segmented,
  Select,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  Upload,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppstoreOutlined,
  DeleteOutlined,
  SearchOutlined,
  UnorderedListOutlined,
  UploadOutlined,
} from '@ant-design/icons';
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
  type ModSyncSide,
  type ModTarget,
} from '@/api/client';
import { CatalogSourceLinks, CatalogSourceSwitch } from '@/components/CatalogSourceSwitch';
import { GameServerCatalogProvider } from '@/components/GameServerCatalogProvider';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { ModCatalogInstallControls } from '@/components/ModCatalogInstallControls';
import { ModSideBadge } from '@/components/ModSideBadge';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { useModal } from '@/hooks/useModal';
import {
  gameServerTypeLabelText,
  pluginLoaderForServerType,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import { attachCatalogPartners } from '@/lib/catalogPartners';
import { cachedGetModProject } from '@/lib/modCatalogCache';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { formatCompactCount } from '@/lib/formatCompactCount';
import {
  catalogCardItem,
  mergeCatalogCardsByName,
  preferredCatalogItem,
  type CatalogCard,
} from '@/lib/mergeCatalogCards';
import {
  contentKindHasSide,
  contentTargetFromPath,
  gameServerInstallSide,
  instanceResourceContentTarget,
  instanceResourceModTarget,
  isCatalogItemOnServer,
  modSyncSide,
  needsServerRestartAfterSync,
} from '@/lib/modSync';
import { useGameServerContentViewMode } from '@/lib/installedResourcesView';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';
import './InstanceResourcesPanel.css';
import './GameServerContentPanel.css';

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

function matchResource(
  file: GameServerFileEntry,
  resources: InstanceResource[],
): InstanceResource | undefined {
  const name = file.name.toLowerCase();
  return resources.find((item) => item.filename.toLowerCase() === name);
}

function resourceFileOnDisk(
  resource: Pick<InstanceResource, 'filename'>,
  files: GameServerFileEntry[],
): boolean {
  const filename = (resource.filename ?? '').toLowerCase();
  return Boolean(filename) && files.some((file) => !file.dir && file.name.toLowerCase() === filename);
}

function formatFileSize(size?: number): string {
  if (size == null || size <= 0) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function projectTypeForKind(kind: GameServerContentKind): ModProjectType {
  return kind;
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
    case 'resourcepack':
      return Promise.all([
        api.listVpsGameServerResourcepacks(vpsId, gameServerId),
        api.listVpsGameServerClientResourcepacks(vpsId, gameServerId),
      ]).then(([serverRes, clientRes]) => ({
        items: [...(serverRes.items ?? []), ...(clientRes.items ?? [])],
      }));
    case 'shader':
      return Promise.all([
        api.listVpsGameServerShaders(vpsId, gameServerId),
        api.listVpsGameServerClientShaders(vpsId, gameServerId),
      ]).then(([serverRes, clientRes]) => ({
        items: [...(serverRes.items ?? []), ...(clientRes.items ?? [])],
      }));
    default:
      return Promise.all([
        api.listVpsGameServerMods(vpsId, gameServerId),
        api.listVpsGameServerClientMods(vpsId, gameServerId),
      ]).then(([modsRes, clientRes]) => ({
        items: [...(modsRes.items ?? []), ...(clientRes.items ?? [])],
      }));
  }
}

function installedModSide(
  file: GameServerFileEntry,
  resource: InstanceResource | undefined,
  kind: GameServerContentKind,
): ModSyncSide {
  const override = resource?.side_override?.trim();
  if (override === 'client' || override === 'server' || override === 'both') {
    return override;
  }
  const target = contentTargetFromPath(file.path);
  if (target === 'client-mods' || target === 'client-resourcepacks' || target === 'client-shaders') {
    return 'client';
  }
  return kind === 'mod' ? 'both' : 'server';
}

function sideFolderKind(side: ModSyncSide): 'client' | 'server' {
  return side === 'client' ? 'client' : 'server';
}

function deleteContent(
  kind: GameServerContentKind,
  vpsId: string,
  gameServerId: string,
  filename: string,
  modTarget?: ModTarget,
) {
  switch (kind) {
    case 'plugin':
      return api.deleteVpsGameServerPlugin(vpsId, gameServerId, { filename });
    case 'datapack':
      return api.deleteVpsGameServerDatapack(vpsId, gameServerId, { filename });
    case 'resourcepack':
      return api.deleteVpsGameServerResourcepack(vpsId, gameServerId, {
        filename,
        mod_target: modTarget,
      });
    case 'shader':
      return api.deleteVpsGameServerShader(vpsId, gameServerId, {
        filename,
        mod_target: modTarget,
      });
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
  const modal = useModal();
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
  const [detailCard, setDetailCard] = useState<CatalogCard | null>(null);
  const [detailSource, setDetailSource] = useState<ModSource>('modrinth');
  const [cardSourceByKey, setCardSourceByKey] = useState<Partial<Record<string, ModSource>>>({});
  const [uploading, setUploading] = useState(false);
  const [deletingPath, setDeletingPath] = useState<string>();
  const [sideSavingPath, setSideSavingPath] = useState<string>();
  const [uploadSide, setUploadSide] = useState<ModSyncSide>('both');
  const { viewMode, setViewMode } = useGameServerContentViewMode();

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
      const files = (res.items ?? []).filter((item) => !item.dir);
      setInstalled(files);
      setInstalledResources(
        (resourcesRes.items ?? []).filter((resource) => resourceFileOnDisk(resource, files)),
      );
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
          const searched = res.items ?? [];
          setCatalogItems(searched);
          setHasMore(false);
          setOffset(0);
          setCurseforgeEnabled(res.curseforge_enabled ?? false);
          if (sourceFilter === 'all' && searched.length > 0) {
            const enriched = await attachCatalogPartners(searched, {
              loader,
              mcVersion,
              type: projectType,
            });
            if (!cancelled) setCatalogItems(enriched);
          }
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
        setOffset(PAGE_SIZE);
        setCurseforgeEnabled(res.curseforge_enabled ?? false);
        if (sourceFilter === 'all' && nextItems.length > 0) {
          const enriched = await attachCatalogPartners(nextItems, {
            loader,
            mcVersion,
            type: projectType,
          });
          if (!cancelled) setCatalogItems(enriched);
        }
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
      setCatalogItems((prev) => {
        const combined = [...prev, ...nextItems];
        if (sourceFilter === 'all' && nextItems.length > 0) {
          void attachCatalogPartners(combined, {
            loader,
            mcVersion,
            type: projectType,
          }).then(setCatalogItems);
        }
        return combined;
      });
      setHasMore(res.has_more ?? false);
      setOffset((prev) => prev + PAGE_SIZE);
      setCurseforgeEnabled(res.curseforge_enabled ?? false);
    } catch (e) {
      message.error(formatModCatalogError(e, t, `${i18nPrefix}.browseFailed`));
    } finally {
      setLoadingMore(false);
    }
  };

  const isSearchMode = appliedSearch.trim().length > 0;

  const isItemOnServer = useCallback(
    (item: ModCatalogItem) => {
      if (isCatalogItemOnServer(item, installed)) {
        return true;
      }
      return installedResources.some(
        (resource) =>
          resource.source === item.source &&
          resource.project_id === item.id &&
          resourceFileOnDisk(resource, installed),
      );
    },
    [installed, installedResources],
  );

  const catalogCards = useMemo(
    () => mergeCatalogCardsByName(catalogItems, sourceFilter),
    [catalogItems, sourceFilter],
  );

  const itemForCard = useCallback(
    (card: CatalogCard) => catalogCardItem(card, cardSourceByKey[card.key]),
    [cardSourceByKey],
  );

  const isCardOnServer = useCallback(
    (card: CatalogCard) => card.items.some((item) => isItemOnServer(item)),
    [isItemOnServer],
  );

  const installedProjectIds = useMemo(() => {
    const keys = new Set<string>();
    for (const resource of installedResources) {
      if (resource.project_id && resourceFileOnDisk(resource, installed)) {
        keys.add(`${resource.source}:${resource.project_id}`);
      }
    }
    for (const item of catalogItems) {
      if (isItemOnServer(item)) {
        keys.add(`${item.source}:${item.id}`);
      }
    }
    return keys;
  }, [catalogItems, installed, installedResources, isItemOnServer]);

  const visibleCatalogCards = useMemo(() => {
    if (showInstalledOnly) {
      return catalogCards.filter((card) => isCardOnServer(card));
    }
    return catalogCards.filter((card) => !isCardOnServer(card));
  }, [catalogCards, isCardOnServer, showInstalledOnly]);

  const setCardSource = (card: CatalogCard, source: ModSource) => {
    setCardSourceByKey((prev) => ({ ...prev, [card.key]: source }));
    if (detailCard?.key === card.key) {
      setDetailSource(source);
    }
  };

  const openDetail = useCallback((card: CatalogCard) => {
    const source = cardSourceByKey[card.key] ?? preferredCatalogItem(card.items).source;
    setDetailCard(card);
    setDetailSource(source);
    setDetailOpen(true);
    for (const item of card.items) {
      if (item.source === 'upload') continue;
      void cachedGetModProject(item.source, item.id)
        .then((full) => {
          setDetailCard((prev) => {
            if (!prev || prev.key !== card.key) return prev;
            return {
              ...prev,
              items: prev.items.map((row) =>
                row.source === item.source && row.id === item.id ? { ...row, ...full } : row,
              ),
            };
          });
        })
        .catch(() => {
          /* keep the catalog row */
        });
    }
  }, [cardSourceByKey]);

  const filteredInstalled = useMemo(() => {
    const query = appliedInstalledSearch.trim().toLowerCase();
    if (!query) return installed;
    return installed.filter((item) => {
      const resource = matchResource(item, installedResources);
      const name = item.name.toLowerCase();
      const path = item.path.toLowerCase();
      const title = (resource?.project_name ?? '').toLowerCase();
      return name.includes(query) || path.includes(query) || title.includes(query);
    });
  }, [appliedInstalledSearch, installed, installedResources]);

  const promptRestart = () => {
    modal.confirm({
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

  const handleDelete = async (row: GameServerFileEntry) => {
    setDeletingPath(row.path);
    try {
      await deleteContent(kind, vpsId, gameServerId, row.name, contentTargetFromPath(row.path));
      message.success(t('gameServerDetail.content.deleteCompleted'));
      void loadInstalled();
      if (needsServerRestartAfterSync(contentTargetFromPath(row.path))) {
        promptRestart();
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.content.deleteFailed'));
    } finally {
      setDeletingPath(undefined);
    }
  };

  const handleUpload = async (file: File) => {
    if (kind !== 'mod') return false;
    setUploading(true);
    try {
      const side = gameServerInstallSide(uploadSide);
      await api.uploadGameServerMod(
        vpsId,
        gameServerId,
        file,
        instanceResourceModTarget({ side_override: side, resource_type: 'mod' }),
        side,
      );
      message.success(t('gameServerDetail.content.uploadCompleted'));
      void loadInstalled();
      if (needsServerRestartAfterSync(instanceResourceModTarget({ side_override: side, resource_type: 'mod' }))) {
        promptRestart();
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.content.uploadFailed'));
    } finally {
      setUploading(false);
    }
    return false;
  };

  const handleSideChange = async (row: GameServerFileEntry, side: ModSyncSide) => {
    const resource = matchResource(row, installedResources);
    const previous = installedModSide(row, resource, kind);
    setSideSavingPath(row.path);
    try {
      await api.patchGameServerResource(vpsId, gameServerId, {
        filename: row.name,
        resource_type: projectType,
        side_override: side,
      });
      message.success(t('qxmods.side.saved'));
      void loadInstalled();
      if (sideFolderKind(previous) !== sideFolderKind(side)) {
        promptRestart();
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.side.saveFailed'));
    } finally {
      setSideSavingPath(undefined);
    }
  };

  const sourceOptions = useMemo(() => {
    const options = [
      { value: 'all', label: t('qxmods.filters.sourceAll') },
      { value: 'modrinth', label: t('qxmods.source.modrinth') },
    ];
    if (kind === 'plugin') {
      options.push(
        { value: 'hangar', label: t('qxmods.source.hangar') },
        { value: 'spigot', label: t('qxmods.source.spigot') },
        { value: 'bukkit', label: t('qxmods.source.bukkit') },
      );
    } else {
      options.push({ value: 'curseforge', label: t('qxmods.source.curseforge') });
    }
    return options;
  }, [kind, t]);

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
    return (
      <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t(`${i18nPrefix}.notSupported`)} />
    );
  }

  if (!agentOnline) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('servers.gameServersAgentRequired')}
      />
    );
  }

  const showCurseforgeUnavailable =
    (sourceFilter === 'curseforge' || sourceFilter === 'bukkit') &&
    catalogLoaded &&
    !curseforgeEnabled;
  const introKey =
    kind === 'plugin'
      ? 'gameServerDetail.content.introPlugin'
      : kind === 'datapack'
        ? 'gameServerDetail.content.introDatapack'
        : kind === 'resourcepack'
          ? 'gameServerDetail.content.introResourcepack'
          : kind === 'shader'
            ? 'gameServerDetail.content.introShader'
            : 'gameServerDetail.content.introMod';
  const loaderLabel = gameServerTypeLabelText(t, serverType);
  const catalogEmptyText = showInstalledOnly
    ? t('qxmods.installed.empty')
    : isSearchMode
      ? t('qxmods.empty')
      : t(`${i18nPrefix}.browseEmpty`);

  const renderInstallControls = (row: ModCatalogItem, layout: 'inline' | 'stacked') => (
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
      layout={layout}
      eagerVersions={layout === 'stacked'}
      onInstalled={() => {
        if (layout === 'stacked') setDetailOpen(false);
        void loadInstalled();
        if (
          needsServerRestartAfterSync(
            contentKindHasSide(kind)
              ? instanceResourceContentTarget({
                  side_override: gameServerInstallSide(modSyncSide(row)),
                  resource_type: projectType,
                })
              : undefined,
          )
        ) {
          promptRestart();
        }
      }}
      onUninstalled={() => void loadInstalled()}
    />
  );

  const viewToggle = (
    <Segmented
      size="small"
      value={viewMode}
      aria-label={t('gameServerDetail.content.viewModeAria')}
      onChange={(value) => setViewMode(value as 'list' | 'cards')}
      options={[
        {
          value: 'list',
          label: (
            <span aria-label={t('gameServerDetail.content.viewList')}>
              <UnorderedListOutlined aria-hidden />
            </span>
          ),
        },
        {
          value: 'cards',
          label: (
            <span aria-label={t('gameServerDetail.content.viewCards')}>
              <AppstoreOutlined aria-hidden />
            </span>
          ),
        },
      ]}
    />
  );

  const sideSelectOptions = [
    { value: 'client', label: t('qxmods.side.client') },
    { value: 'server', label: t('qxmods.side.server') },
    { value: 'both', label: t('qxmods.side.both') },
  ];

  const renderInstalledSideSelect = (row: GameServerFileEntry, resource?: InstanceResource) => {
    if (!contentKindHasSide(kind)) return null;
    return (
      <Select
        size="small"
        className="launcher-resource-side-select"
        loading={sideSavingPath === row.path}
        value={installedModSide(row, resource, kind)}
        options={sideSelectOptions}
        aria-label={t('qxmods.side.editAria')}
        onChange={(value) => void handleSideChange(row, value as ModSyncSide)}
      />
    );
  };

  const catalogColumns: ColumnsType<CatalogCard> = [
    {
      title: '',
      key: 'icon',
      width: 56,
      render: (_, card) => {
        const item = itemForCard(card);
        return (
          <ModCatalogIcon url={item.icon_url} name={item.name} size={44} className="qxmods-catalog-table-icon" />
        );
      },
    },
    {
      title: t('qxmods.catalog.name'),
      dataIndex: 'name',
      key: 'name',
      width: 220,
      ellipsis: true,
      render: (_, card) => {
        const item = itemForCard(card);
        return (
          <div className="qxmods-catalog-name-cell">
            <button type="button" className="game-server-mods-card-name" onClick={() => void openDetail(card)}>
              {card.name}
            </button>
            <div className="qxmods-catalog-name-meta">
              <CatalogSourceSwitch
                items={card.items}
                value={item.source}
                onChange={(source) => setCardSource(card, source)}
              />
              {item.author ? (
                <Typography.Text type="secondary" className="qxmods-catalog-author">
                  {item.author}
                </Typography.Text>
              ) : null}
            </div>
          </div>
        );
      },
    },
    {
      title: t('qxmods.catalog.summary'),
      key: 'summary',
      ellipsis: true,
      responsive: ['md'],
      render: (_, card) => itemForCard(card).summary,
    },
    {
      title: t('qxmods.catalog.side'),
      key: 'side',
      width: 132,
      render: (_, card) => (contentKindHasSide(kind) ? <ModSideBadge item={itemForCard(card)} /> : null),
    },
    {
      title: t('qxmods.catalog.downloads'),
      key: 'downloads',
      width: 96,
      render: (_, card) => {
        const item = itemForCard(card);
        return item.downloads != null ? formatCompactCount(item.downloads) : '—';
      },
    },
    {
      title: t('qxmods.catalog.install'),
      key: 'install',
      width: 260,
      className: 'qxmods-catalog-install-cell',
      render: (_, card) => {
        const item = itemForCard(card);
        return renderInstallControls(item, 'inline');
      },
    },
  ];

  const detailItem = detailCard ? catalogCardItem(detailCard, detailSource) : null;

  return (
    <GameServerCatalogProvider
      kind={kind}
      vpsId={vpsId}
      gameServerId={gameServerId}
      serverType={serverType}
      mcVersion={mcVersion}
    >
      <section className="game-server-mods">
        <header className="game-server-mods-hero">
          <div className="game-server-mods-hero-main">
            <Title level={3} className="game-server-mods-title">
              {t(`${i18nPrefix}.browseTitle`)}
            </Title>
            <Paragraph type="secondary" className="game-server-mods-intro">
              {t(introKey)}
            </Paragraph>
          </div>
          <div className="game-server-mods-chips">
            <span className="game-server-mods-chip">
              Minecraft <strong>{mcVersion}</strong>
            </span>
            <span className="game-server-mods-chip">
              <strong>{loaderLabel}</strong>
            </span>
            <span className="game-server-mods-chip">
              {t('gameServerDetail.content.installedCount', { count: installed.length })}
            </span>
          </div>
        </header>

        <div className="game-server-mods-tabs-row">
          <Segmented
            value={section}
            onChange={(value) => setSection(value as 'installed' | 'catalog')}
            options={[
              {
                value: 'installed',
                label: (
                  <span>
                    {t('gameServerDetail.content.tabInstalled')}{' '}
                    <Badge count={installed.length} showZero size="small" />
                  </span>
                ),
              },
              { value: 'catalog', label: t('gameServerDetail.content.tabCatalog') },
            ]}
          />
          {viewToggle}
        </div>

        {section === 'installed' ? (
          <>
            <div className="game-server-mods-toolbar">
              <div className="game-server-mods-search">
                <Input.Search
                  allowClear
                  enterButton={appliedInstalledSearch ? t('qxmods.applySearch') : t('qxmods.search')}
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
                  onSearch={(value) => setAppliedInstalledSearch(value.trim())}
                  onClear={() => {
                    setInstalledSearchInput('');
                    setAppliedInstalledSearch('');
                  }}
                />
              </div>
              {kind === 'mod' ? (
                <div className="game-server-content-upload-wrapper">
                  <Select
                    size="small"
                    value={uploadSide}
                    options={sideSelectOptions}
                    aria-label={t('gameServerDetail.content.uploadSideAria')}
                    onChange={(value) => setUploadSide(value as ModSyncSide)}
                    className="launcher-resource-side-select"
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
            ) : installed.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t(`${i18nPrefix}.empty`)}
              >
                <Button type="primary" onClick={() => setSection('catalog')}>
                  {t('gameServerDetail.content.openCatalog')}
                </Button>
              </Empty>
            ) : filteredInstalled.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('qxmods.empty')} />
            ) : (
              <ul className={viewMode === 'list' ? 'qxmods-installed-list' : 'game-server-mods-grid'}>
                {filteredInstalled.map((row) => {
                  const resource = matchResource(row, installedResources);
                  const title = resource?.project_name || row.name;
                  const removeButton = (
                    <Popconfirm
                      title={t('gameServerDetail.content.deleteTitle')}
                      description={t('gameServerDetail.content.deleteConfirm', { name: row.name })}
                      okText={t('gameServerDetail.content.deleteAction')}
                      cancelText={t('common.cancel')}
                      okButtonProps={{ danger: true }}
                      onConfirm={() => void handleDelete(row)}
                    >
                      <Button
                        type="text"
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        loading={deletingPath === row.path}
                        aria-label={t('gameServerDetail.content.deleteAction')}
                      />
                    </Popconfirm>
                  );
                  if (viewMode === 'list') {
                    return (
                      <li key={row.path}>
                        <article className="qxmods-installed-item launcher-resources-list-item">
                          <ModCatalogIcon
                            url={resource?.icon_url}
                            name={title}
                            size={40}
                            className="qxmods-installed-icon"
                          />
                          <div className="qxmods-installed-item-content">
                            <div className="qxmods-installed-item-title">
                              <Text strong>{title}</Text>
                              {resource?.source && resource.source !== 'upload' ? (
                                <ModSourceBadge source={resource.source} />
                              ) : (
                                <Tag variant="filled">{t('gameServerDetail.content.uploaded')}</Tag>
                              )}
                            </div>
                            <div className="game-server-mods-card-meta">
                              {resource?.version_number ? (
                                <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--version">
                                  {resource.version_number}
                                </Tag>
                              ) : null}
                              <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--size">
                                {formatFileSize(row.size ?? resource?.file_size)}
                              </Tag>
                            </div>
                            {title !== row.name ? (
                              <Text className="game-server-mods-card-file">{row.name}</Text>
                            ) : null}
                            {renderInstalledSideSelect(row, resource)}
                          </div>
                          <div className="qxmods-installed-item-actions">{removeButton}</div>
                        </article>
                      </li>
                    );
                  }
                  return (
                    <li key={row.path}>
                      <article className="game-server-mods-card">
                        <div className="game-server-mods-card-top">
                          <ModCatalogIcon
                            url={resource?.icon_url}
                            name={title}
                            size={48}
                            className="launcher-resource-card-icon"
                          />
                          <div className="game-server-mods-card-body">
                            <div className="game-server-mods-card-title">
                              <Text strong>{title}</Text>
                              {resource?.source && resource.source !== 'upload' ? (
                                <ModSourceBadge source={resource.source} />
                              ) : (
                                <Tag variant="filled">{t('gameServerDetail.content.uploaded')}</Tag>
                              )}
                            </div>
                            <div className="game-server-mods-card-meta">
                              {resource?.version_number ? (
                                <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--version">
                                  {resource.version_number}
                                </Tag>
                              ) : null}
                              <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--size">
                                {formatFileSize(row.size ?? resource?.file_size)}
                              </Tag>
                            </div>
                            {title !== row.name ? (
                              <Text className="game-server-mods-card-file">{row.name}</Text>
                            ) : null}
                            {renderInstalledSideSelect(row, resource)}
                          </div>
                          {removeButton}
                        </div>
                      </article>
                    </li>
                  );
                })}
              </ul>
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
                <Input.Search
                  allowClear
                  enterButton={appliedSearch ? t('qxmods.applySearch') : t('qxmods.search')}
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
                  onSearch={(value) => setAppliedSearch(value.trim())}
                  onClear={() => {
                    setSearchInput('');
                    setAppliedSearch('');
                  }}
                />
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
            ) : visibleCatalogCards.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={catalogEmptyText} />
            ) : (
              <>
                {viewMode === 'list' ? (
                  <Table
                    className="qxmods-catalog-table qxmods-catalog-table--install"
                    rowKey={(card) => card.key}
                    columns={catalogColumns}
                    dataSource={visibleCatalogCards}
                    loading={catalogLoading && !showInstalledOnly}
                    pagination={false}
                    scroll={{ x: 960 }}
                    tableLayout="fixed"
                    locale={{
                      emptyText: (
                        <Empty
                          image={Empty.PRESENTED_IMAGE_SIMPLE}
                          description={catalogEmptyText}
                        />
                      ),
                    }}
                  />
                ) : (
                  <ul className="game-server-mods-grid">
                    {visibleCatalogCards.map((card) => {
                      const row = itemForCard(card);
                      return (
                        <li key={card.key}>
                          <article className="game-server-mods-card">
                            <div className="game-server-mods-card-top">
                              <ModCatalogIcon
                                url={row.icon_url}
                                name={row.name}
                                size={48}
                                className="launcher-resource-card-icon"
                              />
                              <div className="game-server-mods-card-body">
                                <div className="game-server-mods-card-title">
                                  <button
                                    type="button"
                                    className="game-server-mods-card-name"
                                    onClick={() => void openDetail(card)}
                                  >
                                    {card.name}
                                  </button>
                                  <CatalogSourceSwitch
                                    items={card.items}
                                    value={row.source}
                                    onChange={(source) => setCardSource(card, source)}
                                  />
                                </div>
                                {row.author ? (
                                  <Text type="secondary">
                                    {t('gameServerDetail.content.byAuthor', { author: row.author })}
                                  </Text>
                                ) : null}
                                {row.summary ? (
                                  <p className="game-server-mods-card-summary">{row.summary}</p>
                                ) : null}
                                <div className="game-server-mods-card-meta">
                                  {contentKindHasSide(kind) ? <ModSideBadge item={row} /> : null}
                                  {row.downloads != null ? (
                                    <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--downloads">
                                      {t('gameServerDetail.content.downloadsLabel', {
                                        count: formatCompactCount(row.downloads),
                                      })}
                                    </Tag>
                                  ) : null}
                                </div>
                              </div>
                            </div>
                            <div className="game-server-mods-card-actions" key={`${row.source}:${row.id}`}>
                              {renderInstallControls(row, 'inline')}
                            </div>
                          </article>
                        </li>
                      );
                    })}
                  </ul>
                )}
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
          width={680}
        >
          {detailCard && detailItem ? (
            <div className="game-server-mods-detail">
              <div className="game-server-mods-detail-head">
                <ModCatalogIcon
                  url={detailItem.icon_url}
                  name={detailItem.name}
                  size={72}
                  className="launcher-resource-card-icon"
                />
                <div className="game-server-mods-detail-copy">
                  <div className="game-server-mods-detail-meta">
                    <CatalogSourceSwitch
                      items={detailCard.items}
                      value={detailItem.source}
                      onChange={(source) => setCardSource(detailCard, source)}
                    />
                    {contentKindHasSide(kind) ? <ModSideBadge item={detailItem} /> : null}
                    {detailItem.author ? (
                      <Text type="secondary">
                        {t('gameServerDetail.content.byAuthor', { author: detailItem.author })}
                      </Text>
                    ) : null}
                    {detailItem.downloads != null ? (
                      <Text type="secondary">
                        {t('gameServerDetail.content.downloadsLabel', {
                          count: formatCompactCount(detailItem.downloads),
                        })}
                      </Text>
                    ) : null}
                  </div>
                  {detailItem.summary ? <Paragraph>{detailItem.summary}</Paragraph> : null}
                  <CatalogSourceLinks items={detailCard.items} />
                </div>
              </div>
              <div key={`${detailItem.source}:${detailItem.id}`}>
                {renderInstallControls(detailItem, 'stacked')}
              </div>
            </div>
          ) : null}
        </Modal>
      </section>
    </GameServerCatalogProvider>
  );
}
