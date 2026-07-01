import { useEffect, useMemo, useState } from 'react';
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
import { ArrowLeftOutlined, SearchOutlined } from '@ant-design/icons';
import {
  api,
  type ModCatalogItem,
  type ModCatalogSort,
  type ModCatalogSourceFilter,
  type ModProjectType,
} from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { formatModCatalogError } from '@/lib/modCatalogError';
import './InstanceResourcesPanel.css';

const { Text, Paragraph } = Typography;
const PAGE_SIZE = 20;
const TAB_TYPES: ModProjectType[] = ['mod', 'modpack', 'resourcepack', 'shader'];

export function ModsCatalogPanel() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, basePath } = useInstanceMods();
  const [activeTab, setActiveTab] = useState<ModProjectType>('mod');
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

  const isSearchMode = appliedSearch.trim().length > 0;
  const showCurseforgeUnavailable =
    sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled && !isSearchMode;

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
            loader: instance.loader,
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
          loader: instance.loader,
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
  }, [activeTab, appliedSearch, instance.loader, instance.mc_version, sort, sourceFilter, message, t]);

  const loadMore = async () => {
    if (appliedSearch.trim()) return;
    setLoadingMore(true);
    try {
      const res = await api.browseMods({
        type: activeTab,
        loader: instance.loader,
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
      title: t('qxmods.catalog.name'),
      dataIndex: 'name',
      key: 'name',
      render: (_, item) => (
        <Link to={`${basePath}/catalog/${item.source}/${item.id}`} className="qxmods-catalog-link">
          {item.name}
        </Link>
      ),
    },
    {
      title: t('qxmods.filters.source'),
      key: 'source',
      width: 120,
      render: (_, item) => <ModSourceBadge source={item.source} />,
    },
    {
      title: t('qxmods.catalog.summary'),
      dataIndex: 'summary',
      key: 'summary',
      ellipsis: true,
    },
  ];

  return (
    <section className="instance-resources-panel instance-resources-panel--standalone qxmods-catalog-page">
      <div className="qxmods-page-toolbar">
        <Link to={basePath} className="launcher-instance-detail-back">
          <ArrowLeftOutlined /> {t('qxmods.catalog.backToInstalled')}
        </Link>
      </div>
      <div className="instance-resources-header">
        <Typography.Title level={5} className="instance-resources-title">
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
        options={TAB_TYPES.map((type) => ({ value: type, label: t(`qxmods.tabs.${type}`) }))}
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
        <Alert type="warning" showIcon message={t('qxmods.curseforgeDisabled')} />
      ) : (
        <>
          <Table
            className="qxmods-catalog-table"
            rowKey={(item) => `${item.source}:${item.id}`}
            columns={columns}
            dataSource={items}
            pagination={false}
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
    </section>
  );
}
