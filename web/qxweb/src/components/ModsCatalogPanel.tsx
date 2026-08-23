import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Alert,
  Button,
  Empty,
  Grid,
  Input,
  Segmented,
  Select,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { AppstoreOutlined, SearchOutlined, UnorderedListOutlined } from '@ant-design/icons';
import {
  api,
  type ModCatalogItem,
  type ModCatalogSort,
  type ModCatalogSourceFilter,
  type ModProjectType,
  type ModSource,
  type ModVersion,
  type InstanceResource,
} from '@/api/client';
import { CatalogSourceSwitch } from '@/components/CatalogSourceSwitch';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import {
  ModCatalogInstallControls,
  clearModVersionCache,
} from '@/components/ModCatalogInstallControls';
import { ModSideBadge } from '@/components/ModSideBadge';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { attachCatalogPartners } from '@/lib/catalogPartners';
import { formatCompactCount } from '@/lib/formatCompactCount';
import {
  catalogCardItem,
  mergeCatalogCardsByName,
  type CatalogCard,
} from '@/lib/mergeCatalogCards';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { isCatalogItemInstalledOnInstance, modSupportsServerSync } from '@/lib/modSync';
import {
  catalogLoaderForType,
  launcherCatalogTabs,
} from '@/lib/launcherInstanceCapabilities';
import './InstanceResourcesPanel.css';
import './GameServerContentPanel.css';

const { Text, Paragraph } = Typography;
const PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 400;

export function ModsCatalogPanel() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, basePath, canSync } = useInstanceMods();
  const tabTypes = useMemo(() => launcherCatalogTabs(instance.loader), [instance.loader]);
  const [activeTab, setActiveTab] = useState<ModProjectType>(() => tabTypes[0] ?? 'mod');
  const [sourceFilter, setSourceFilter] = useState<ModCatalogSourceFilter>('all');
  const [sort, setSort] = useState<ModCatalogSort>('downloads');
  const [searchInput, setSearchInput] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [items, setItems] = useState<ModCatalogItem[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [curseforgeEnabled, setCurseforgeEnabled] = useState(false);
  const [catalogLoaded, setCatalogLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [installedResources, setInstalledResources] = useState<InstanceResource[]>([]);
  const [showInstalledOnly, setShowInstalledOnly] = useState(false);
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncSelection, setSyncSelection] = useState<ModSyncSelection | null>(null);
  const [cardSourceByKey, setCardSourceByKey] = useState<Partial<Record<string, ModSource>>>({});
  const screens = Grid.useBreakpoint();
  const [viewMode, setViewMode] = useState<'list' | 'cards'>(screens.md === false ? 'cards' : 'list');

  const refreshInstalled = useCallback(async () => {
    try {
      const res = await api.listInstanceResources(instance.id);
      const resources = res.items ?? [];
      setInstalledResources(resources);
      return resources;
    } catch {
      setInstalledResources([]);
      return [];
    }
  }, [instance.id]);

  const handleInstalled = useCallback(
    async (item: ModCatalogItem, version: ModVersion) => {
      await refreshInstalled();
      const projectType = activeTab;
      if (canSync && modSupportsServerSync(item) && projectType === 'mod') {
        setSyncSelection({
          source: item.source as ModSource,
          projectId: item.id,
          projectName: item.name,
          version,
        });
        setSyncOpen(true);
      }
    },
    [activeTab, canSync, refreshInstalled],
  );

  const isSearchMode = appliedSearch.trim().length > 0;
  const showCurseforgeUnavailable =
    sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled;

  const catalogLoader = catalogLoaderForType(instance.loader, activeTab);

  useEffect(() => {
    void refreshInstalled();
  }, [refreshInstalled]);

  useEffect(() => {
    if (!tabTypes.includes(activeTab)) {
      setActiveTab(tabTypes[0] ?? 'mod');
    }
  }, [activeTab, tabTypes]);

  useEffect(() => {
    clearModVersionCache();
  }, [activeTab, appliedSearch, catalogLoader, instance.mc_version, sort, sourceFilter]);

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
    let cancelled = false;
    setCatalogLoaded(false);
    void (async () => {
      setLoading(true);
      setLoadingMore(false);
      try {
        if (appliedSearch.trim()) {
          const res = await api.searchMods({
            q: appliedSearch.trim(),
            type: activeTab,
            loader: catalogLoader,
            mc_version: instance.mc_version,
            source: sourceFilter,
            limit: PAGE_SIZE,
          });
          if (cancelled) return;
          const searched = res.items ?? [];
          setItems(searched);
          setHasMore(false);
          setOffset(0);
          setCurseforgeEnabled(res.curseforge_enabled ?? false);
          if (sourceFilter === 'all' && searched.length > 0) {
            const enriched = await attachCatalogPartners(searched, {
              loader: catalogLoader,
              mcVersion: instance.mc_version,
              type: activeTab,
            });
            if (!cancelled) setItems(enriched);
          }
          return;
        }

        const res = await api.browseMods({
          type: activeTab,
          loader: catalogLoader,
          mc_version: instance.mc_version,
          source: sourceFilter,
          sort,
          limit: PAGE_SIZE,
          offset: 0,
        });
        if (cancelled) return;
        const nextItems = res.items ?? [];
        setItems(nextItems);
        setHasMore(res.has_more ?? false);
        setOffset(PAGE_SIZE);
        setCurseforgeEnabled(res.curseforge_enabled ?? false);
        if (sourceFilter === 'all' && nextItems.length > 0) {
          const enriched = await attachCatalogPartners(nextItems, {
            loader: catalogLoader,
            mcVersion: instance.mc_version,
            type: activeTab,
          });
          if (!cancelled) setItems(enriched);
        }
      } catch (e) {
        if (cancelled) return;
        message.error(
          formatModCatalogError(
            e,
            t,
            appliedSearch.trim() ? 'qxmods.searchFailed' : 'qxmods.browseFailed',
          ),
        );
        setItems([]);
        setHasMore(false);
        setOffset(0);
      } finally {
        if (!cancelled) {
          setLoading(false);
          setCatalogLoaded(true);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeTab, appliedSearch, catalogLoader, instance.mc_version, sort, sourceFilter, message, t]);

  const loadMore = async () => {
    if (appliedSearch.trim()) return;
    setLoadingMore(true);
    try {
      const res = await api.browseMods({
        type: activeTab,
        loader: catalogLoader,
        mc_version: instance.mc_version,
        source: sourceFilter,
        sort,
        limit: PAGE_SIZE,
        offset,
      });
      const nextItems = res.items ?? [];
      setItems((prev) => {
        const combined = [...prev, ...nextItems];
        if (sourceFilter === 'all' && nextItems.length > 0) {
          void attachCatalogPartners(combined, {
            loader: catalogLoader,
            mcVersion: instance.mc_version,
            type: activeTab,
          }).then(setItems);
        }
        return combined;
      });
      setHasMore(res.has_more ?? false);
      setOffset((prev) => prev + PAGE_SIZE);
      setCurseforgeEnabled(res.curseforge_enabled ?? false);
    } catch (e) {
      message.error(formatModCatalogError(e, t, 'qxmods.browseFailed'));
    } finally {
      setLoadingMore(false);
    }
  };

  const catalogByKey = useMemo(() => {
    const map = new Map<string, ModCatalogItem>();
    for (const item of items) {
      map.set(`${item.source}:${item.id}`, item);
    }
    return map;
  }, [items]);

  const installedProjectIds = useMemo(() => {
    const keys = new Set<string>();
    for (const resource of installedResources) {
      if (resource.project_id) {
        keys.add(`${resource.source}:${resource.project_id}`);
      }
    }
    for (const item of items) {
      if (isCatalogItemInstalledOnInstance(item, installedResources)) {
        keys.add(`${item.source}:${item.id}`);
      }
    }
    return keys;
  }, [installedResources, items]);

  const visibleItems = useMemo(() => {
    if (showInstalledOnly) {
      const query = appliedSearch.trim().toLowerCase();
      return installedResources
        .filter((resource) => {
          if (resource.resource_type !== activeTab) {
            return false;
          }
          if (sourceFilter !== 'all' && resource.source !== sourceFilter) {
            return false;
          }
          const label = (resource.project_name || resource.filename).toLowerCase();
          if (query && !label.includes(query) && !resource.filename.toLowerCase().includes(query)) {
            return false;
          }
          return true;
        })
        .map((resource) => {
          if (resource.project_id) {
            const fromCatalog = catalogByKey.get(`${resource.source}:${resource.project_id}`);
            if (fromCatalog) {
              return fromCatalog;
            }
            return {
              id: resource.project_id,
              source: resource.source,
              slug: resource.project_id,
              name: resource.project_name,
              icon_url: resource.icon_url,
              downloads: resource.downloads,
              project_type: resource.resource_type,
              external_url: '',
            } satisfies ModCatalogItem;
          }
          const matched = items.find((item) => isCatalogItemInstalledOnInstance(item, [resource]));
          if (matched) {
            return matched;
          }
          return {
            id: resource.filename,
            source: resource.source,
            slug: resource.filename,
            name: resource.project_name || resource.filename,
            icon_url: resource.icon_url,
            downloads: resource.downloads,
            project_type: resource.resource_type,
            external_url: '',
          } satisfies ModCatalogItem;
        });
    }
    return items.filter((item) => !isCatalogItemInstalledOnInstance(item, installedResources));
  }, [
    activeTab,
    appliedSearch,
    catalogByKey,
    installedResources,
    items,
    showInstalledOnly,
    sourceFilter,
  ]);

  const visibleCards = useMemo(
    () => mergeCatalogCardsByName(visibleItems, sourceFilter),
    [sourceFilter, visibleItems],
  );

  const itemForCard = useCallback(
    (card: CatalogCard) => catalogCardItem(card, cardSourceByKey[card.key]),
    [cardSourceByKey],
  );

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

  const columns: ColumnsType<CatalogCard> = [
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
      className: 'qxmods-catalog-name-col',
      render: (_, card) => {
        const item = itemForCard(card);
        return (
          <div className="qxmods-catalog-name-cell">
            <Link
              to={`${basePath}/catalog/${item.source}/${item.id}`}
              state={card.items.length > 1 ? { catalogSiblings: card.items } : undefined}
              className="qxmods-catalog-link"
            >
              {card.name}
            </Link>
            <div className="qxmods-catalog-name-meta">
              <CatalogSourceSwitch
                items={card.items}
                value={item.source}
                onChange={(source) => setCardSourceByKey((prev) => ({ ...prev, [card.key]: source }))}
              />
              {item.author ? (
                <Text type="secondary" className="qxmods-catalog-author">
                  {item.author}
                </Text>
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
      className: 'qxmods-catalog-summary-col',
      responsive: ['md'],
      render: (_, card) => itemForCard(card).summary,
    },
    {
      title: t('qxmods.catalog.side'),
      key: 'side',
      width: 132,
      className: 'qxmods-catalog-side-col',
      render: (_, card) => (activeTab === 'mod' ? <ModSideBadge item={itemForCard(card)} /> : null),
    },
    {
      title: t('qxmods.catalog.downloads'),
      key: 'downloads',
      width: 96,
      className: 'qxmods-catalog-downloads-col',
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
        return (
          <ModCatalogInstallControls
            source={item.source as ModSource}
            projectId={item.id}
            projectName={item.name}
            projectType={activeTab}
            iconUrl={item.icon_url}
            downloads={item.downloads}
            clientSide={item.client_side}
            serverSide={item.server_side}
            loader={catalogLoader}
            mcVersion={instance.mc_version}
            installedProjectIds={installedProjectIds}
            layout="inline"
            selectClassName="qxmods-install-version-select--table"
            onInstalled={(version) => handleInstalled(item, version)}
            onUninstalled={() => void refreshInstalled()}
          />
        );
      },
    },
  ];

  return (
    <section className="instance-resources-panel instance-resources-panel--standalone qxmods-catalog-page">
      <div className="instance-resources-header">
        <Typography.Title level={4} className="instance-resources-title">
          {t('qxmods.catalog.title')}
        </Typography.Title>
        <Text type="secondary" className="instance-resources-brand">
          {t('qxmods.brand')}
        </Text>
      </div>
      <Paragraph type="secondary" className="qxmods-catalog-intro">
        {t('qxmods.catalogIntro')}
      </Paragraph>
      <Segmented
        className="qxmods-type-segmented"
        value={activeTab}
        options={tabTypes.map((type) => ({ value: type, label: t(`qxmods.tabs.${type}`) }))}
        onChange={(value) => setActiveTab(value as ModProjectType)}
      />
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
            prefix={<SearchOutlined />}
            placeholder={t('qxmods.searchFilterPlaceholder')}
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
        </div>
      </div>
      <Paragraph type="secondary" className="qxmods-filter-context">
        {t('qxmods.filterContext', {
          mcVersion: instance.mc_version,
          loader: instance.loader,
        })}
      </Paragraph>
      {loading && items.length === 0 && !showInstalledOnly ? (
        <div className="qxmods-loading">
          <Spin />
        </div>
      ) : showCurseforgeUnavailable ? (
        <Alert type="warning" showIcon title={t('qxmods.curseforgeDisabled')} />
      ) : (
        <>
          {viewMode === 'list' ? (
            <Table
              className="qxmods-catalog-table qxmods-catalog-table--install"
              rowKey={(card) => card.key}
              columns={columns}
              dataSource={visibleCards}
              loading={loading && !showInstalledOnly}
              pagination={false}
              scroll={{ x: 960 }}
              tableLayout="fixed"
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={
                      showInstalledOnly
                        ? t('qxmods.installed.empty')
                        : isSearchMode
                          ? t('qxmods.empty')
                          : t('qxmods.catalogEmpty')
                    }
                  />
                ),
              }}
            />
          ) : (
            <ul className="game-server-mods-grid">
              {visibleCards.map((card) => {
                const item = itemForCard(card);
                return (
                  <li key={card.key}>
                    <article className="game-server-mods-card">
                      <div className="game-server-mods-card-top">
                        <ModCatalogIcon
                          url={item.icon_url}
                          name={item.name}
                          size={48}
                          className="launcher-resource-card-icon"
                        />
                        <div className="game-server-mods-card-body">
                          <div className="game-server-mods-card-title">
                            <Link
                              to={`${basePath}/catalog/${item.source}/${item.id}`}
                              state={card.items.length > 1 ? { catalogSiblings: card.items } : undefined}
                              className="game-server-mods-card-name"
                            >
                              {card.name}
                            </Link>
                            <CatalogSourceSwitch
                              items={card.items}
                              value={item.source}
                              onChange={(source) =>
                                setCardSourceByKey((prev) => ({ ...prev, [card.key]: source }))
                              }
                            />
                          </div>
                          {item.author ? (
                            <Text type="secondary">{item.author}</Text>
                          ) : null}
                          {item.summary ? (
                            <p className="game-server-mods-card-summary">{item.summary}</p>
                          ) : null}
                          <div className="game-server-mods-card-meta">
                            {activeTab === 'mod' ? <ModSideBadge item={item} /> : null}
                            {item.downloads != null ? (
                              <Tag variant="filled">
                                {formatCompactCount(item.downloads)}
                              </Tag>
                            ) : null}
                          </div>
                        </div>
                      </div>
                      <div className="game-server-mods-card-actions">
                        <ModCatalogInstallControls
                          source={item.source as ModSource}
                          projectId={item.id}
                          projectName={item.name}
                          projectType={activeTab}
                          iconUrl={item.icon_url}
                          downloads={item.downloads}
                          clientSide={item.client_side}
                          serverSide={item.server_side}
                          loader={catalogLoader}
                          mcVersion={instance.mc_version}
                          installedProjectIds={installedProjectIds}
                          layout="inline"
                          onInstalled={(version) => handleInstalled(item, version)}
                          onUninstalled={() => void refreshInstalled()}
                        />
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
                {t('qxmods.loadMore')}
              </Button>
            </div>
          ) : null}
        </>
      )}
      <Paragraph type="secondary" className="qxmods-attribution">
        {t('qxmods.attribution')}
        {!curseforgeEnabled ? ` ${t('qxmods.curseforgeDisabled')}` : ''}
      </Paragraph>
      <ModSyncModal
        open={syncOpen}
        selection={syncSelection}
        instanceLoader={instance.loader}
        instanceMcVersion={instance.mc_version}
        installedResources={installedResources}
        onClose={() => setSyncOpen(false)}
      />
    </section>
  );
}
