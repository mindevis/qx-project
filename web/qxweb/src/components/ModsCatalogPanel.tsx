import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Alert,
  Button,
  Input,
  Segmented,
  Select,
  Spin,
  Table,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { SearchOutlined } from '@ant-design/icons';
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
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import {
  ModCatalogInstallControls,
  clearModVersionCache,
} from '@/components/ModCatalogInstallControls';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ModSideBadge } from '@/components/ModSideBadge';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { modSupportsServerSync } from '@/lib/modSync';
import {
  catalogLoaderForType,
  launcherCatalogTabs,
} from '@/lib/launcherInstanceCapabilities';
import './InstanceResourcesPanel.css';

const { Text, Paragraph } = Typography;
const PAGE_SIZE = 20;

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
  const [installedProjectIds, setInstalledProjectIds] = useState<Set<string>>(new Set());
  const [installedResources, setInstalledResources] = useState<InstanceResource[]>([]);
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncSelection, setSyncSelection] = useState<ModSyncSelection | null>(null);

  const refreshInstalled = useCallback(async () => {
    try {
      const res = await api.listInstanceResources(instance.id);
      const resources = res.items ?? [];
      setInstalledResources(resources);
      setInstalledProjectIds(
        new Set(
          resources
            .filter((r) => r.project_id)
            .map((r) => `${r.source}:${r.project_id}`),
        ),
      );
      return resources;
    } catch {
      setInstalledResources([]);
      setInstalledProjectIds(new Set());
      return [];
    }
  }, [instance.id]);

  const handleInstalled = useCallback(
    async (item: ModCatalogItem, version: ModVersion) => {
      await refreshInstalled();
      const projectType = item.project_type ?? activeTab;
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
    sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled && !isSearchMode;

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
    let cancelled = false;
    setCatalogLoaded(false);
    void (async () => {
      setLoading(true);
      setLoadingMore(false);
      try {
        if (!appliedSearch.trim() && sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled) {
          setItems([]);
          setHasMore(false);
          setOffset(0);
          return;
        }

        if (appliedSearch.trim()) {
          const res = await api.searchMods({
            q: appliedSearch.trim(),
            type: activeTab,
            loader: catalogLoader,
            mc_version: instance.mc_version,
            limit: PAGE_SIZE,
          });
          if (cancelled) return;
          setItems(res.items ?? []);
          setHasMore(false);
          setOffset(0);
          setCurseforgeEnabled(res.curseforge_enabled ?? false);
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
        setOffset(nextItems.length);
        setCurseforgeEnabled(res.curseforge_enabled ?? false);
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
      setItems((prev) => [...prev, ...nextItems]);
      setHasMore(res.has_more ?? false);
      setOffset((prev) => prev + nextItems.length);
      setCurseforgeEnabled(res.curseforge_enabled ?? false);
    } catch (e) {
      message.error(formatModCatalogError(e, t, 'qxmods.browseFailed'));
    } finally {
      setLoadingMore(false);
    }
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

  const columns: ColumnsType<ModCatalogItem> = [
    {
      title: '',
      key: 'icon',
      width: 56,
      render: (_, item) => (
        <ModCatalogIcon url={item.icon_url} name={item.name} size={44} className="qxmods-catalog-table-icon" />
      ),
    },
    {
      title: t('qxmods.catalog.name'),
      dataIndex: 'name',
      key: 'name',
      width: 220,
      ellipsis: true,
      className: 'qxmods-catalog-name-col',
      render: (_, item) => (
        <div className="qxmods-catalog-name-cell">
          <Link to={`${basePath}/catalog/${item.source}/${item.id}`} className="qxmods-catalog-link">
            {item.name}
          </Link>
          <div className="qxmods-catalog-name-meta">
            <ModSourceBadge source={item.source} />
          </div>
        </div>
      ),
    },
    {
      title: t('qxmods.catalog.summary'),
      dataIndex: 'summary',
      key: 'summary',
      ellipsis: true,
      className: 'qxmods-catalog-summary-col',
      responsive: ['md'],
    },
    {
      title: t('qxmods.catalog.side'),
      key: 'side',
      width: 132,
      className: 'qxmods-catalog-side-col',
      render: (_, item) => (activeTab === 'mod' ? <ModSideBadge item={item} /> : null),
    },
    {
      title: t('qxmods.catalog.install'),
      key: 'install',
      width: 400,
      className: 'qxmods-catalog-install-cell',
      render: (_, item) => (
        <ModCatalogInstallControls
          source={item.source as ModSource}
          projectId={item.id}
          projectName={item.name}
          projectType={item.project_type ?? activeTab}
          iconUrl={item.icon_url}
          downloads={item.downloads}
          loader={catalogLoader}
          mcVersion={instance.mc_version}
          installedProjectIds={installedProjectIds}
          layout="inline"
          selectClassName="qxmods-install-version-select--table"
          onInstalled={(version) => handleInstalled(item, version)}
          onUninstalled={() => void refreshInstalled()}
        />
      ),
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
              disabled={isSearchMode}
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
        </div>
        <div className="qxmods-search-filter">
          <Input
            allowClear
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
            <Button type="link" onClick={() => {
              setSearchInput('');
              setAppliedSearch('');
            }}>
              {t('qxmods.clearSearch')}
            </Button>
          ) : null}
        </div>
      </div>
      <Paragraph type="secondary" className="qxmods-filter-context">
        {t('qxmods.filterContext', {
          mcVersion: instance.mc_version,
          loader: instance.loader,
        })}
      </Paragraph>
      {loading ? (
        <div className="qxmods-loading">
          <Spin />
        </div>
      ) : showCurseforgeUnavailable ? (
        <Alert type="warning" showIcon title={t('qxmods.curseforgeDisabled')} />
      ) : (
        <>
          <Table
            className="qxmods-catalog-table qxmods-catalog-table--install"
            rowKey={(item) => `${item.source}:${item.id}`}
            columns={columns}
            dataSource={items}
            pagination={false}
            scroll={{ x: 960 }}
            tableLayout="fixed"
            locale={{ emptyText: isSearchMode ? t('qxmods.empty') : t('qxmods.catalogEmpty') }}
          />
          {!isSearchMode && hasMore ? (
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
