import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Drawer,
  Empty,
  Input,
  List,
  Select,
  Space,
  Spin,
  Tabs,
  Typography,
} from 'antd';
import { CloudSyncOutlined, LinkOutlined, SearchOutlined } from '@ant-design/icons';
import {
  api,
  type LauncherInstance,
  type ModCatalogItem,
  type ModCatalogSort,
  type ModCatalogSourceFilter,
  type ModProjectType,
  type ModVersion,
} from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { isModdedLauncherLoader } from '@/lib/isModdedLoader';
import { modSupportsServerSync } from '@/lib/modSync';
import './InstanceResourcesPanel.css';

const { Text, Paragraph, Title } = Typography;

type InstanceResourcesPanelProps = {
  instance: LauncherInstance;
  canSync: boolean;
  layout?: 'embedded' | 'standalone';
};

const TAB_TYPES: ModProjectType[] = ['mod', 'modpack', 'resourcepack', 'shader'];
const PAGE_SIZE = 20;

export function InstanceResourcesPanel({
  instance,
  canSync,
  layout = 'embedded',
}: InstanceResourcesPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
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
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailItem, setDetailItem] = useState<ModCatalogItem | null>(null);
  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [selectedVersionId, setSelectedVersionId] = useState<string>();
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncSelection, setSyncSelection] = useState<ModSyncSelection | null>(null);

  const modded = isModdedLauncherLoader(instance.loader);
  const isSearchMode = appliedSearch.trim().length > 0;
  const showCurseforgeUnavailable =
    sourceFilter === 'curseforge' && catalogLoaded && !curseforgeEnabled && !isSearchMode;

  useEffect(() => {
    if (!modded) return;

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
          e instanceof Error
            ? e.message
            : appliedSearch.trim()
              ? t('qxmods.searchFailed')
              : t('qxmods.browseFailed'),
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
  }, [activeTab, appliedSearch, instance.loader, instance.mc_version, modded, sort, sourceFilter, message, t]);

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
      message.error(e instanceof Error ? e.message : t('qxmods.browseFailed'));
    } finally {
      setLoadingMore(false);
    }
  };

  const applySearch = () => {
    setAppliedSearch(searchInput.trim());
  };

  const clearSearch = () => {
    setSearchInput('');
    setAppliedSearch('');
  };

  const openDetail = async (item: ModCatalogItem) => {
    setDetailItem(item);
    setDetailOpen(true);
    setVersions([]);
    setSelectedVersionId(undefined);
    setVersionsLoading(true);
    try {
      const res = await api.listModVersions(item.source, item.id, {
        loader: instance.loader,
        mc_version: instance.mc_version,
      });
      const list = res.items ?? [];
      setVersions(list);
      setSelectedVersionId(list[0]?.id);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.versionsFailed'));
    } finally {
      setVersionsLoading(false);
    }
  };

  const selectedVersion = useMemo(
    () => versions.find((v) => v.id === selectedVersionId),
    [selectedVersionId, versions],
  );

  const showSyncButton =
    canSync &&
    detailItem != null &&
    selectedVersion != null &&
    modSupportsServerSync(detailItem) &&
    activeTab === 'mod';

  const handleSyncClick = () => {
    if (!detailItem || !selectedVersion) return;
    setSyncSelection({
      source: detailItem.source,
      projectId: detailItem.id,
      projectName: detailItem.name,
      version: selectedVersion,
    });
    setSyncOpen(true);
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

  if (!modded) {
    return null;
  }

  const catalogList = (
    <>
      {loading ? (
        <div className="qxmods-loading">
          <Spin />
        </div>
      ) : showCurseforgeUnavailable ? (
        <Alert type="warning" showIcon message={t('qxmods.curseforgeDisabled')} />
      ) : items.length === 0 ? (
        <Empty description={isSearchMode ? t('qxmods.empty') : t('qxmods.catalogEmpty')} />
      ) : (
        <>
          <List
            className="qxmods-results"
            dataSource={items}
            renderItem={(item) => (
              <List.Item
                key={`${item.source}:${item.id}`}
                className="qxmods-result-item"
                actions={[
                  <Button key="open" type="link" onClick={() => void openDetail(item)}>
                    {t('common.open')}
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  avatar={
                    item.icon_url ? (
                      <img src={item.icon_url} alt={item.name} className="qxmods-result-icon" />
                    ) : (
                      <span className="qxmods-result-icon qxmods-result-icon--placeholder" />
                    )
                  }
                  title={
                    <Space wrap size="small">
                      <span>{item.name}</span>
                      <ModSourceBadge source={item.source} />
                    </Space>
                  }
                  description={item.summary}
                />
              </List.Item>
            )}
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
    </>
  );

  const tabItems = TAB_TYPES.map((type) => ({
    key: type,
    label: t(`qxmods.tabs.${type}`),
    children: (
      <div className="qxmods-tab-panel">
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
              onPressEnter={applySearch}
              onClear={clearSearch}
            />
            <Button onClick={applySearch} disabled={!searchInput.trim() && !appliedSearch}>
              {appliedSearch ? t('qxmods.applySearch') : t('qxmods.search')}
            </Button>
            {appliedSearch ? (
              <Button type="link" onClick={clearSearch}>
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
        <Paragraph type="secondary" className="qxmods-attribution">
          {t('qxmods.attribution')}
          {!curseforgeEnabled ? ` ${t('qxmods.curseforgeDisabled')}` : ''}
        </Paragraph>
        {catalogList}
      </div>
    ),
  }));

  return (
    <section
      className={`instance-resources-panel${layout === 'standalone' ? ' instance-resources-panel--standalone' : ''}`}
      aria-label={t('qxmods.sectionTitle')}
    >
      <div className="instance-resources-header">
        <Title level={5} className="instance-resources-title">
          {t('qxmods.sectionTitle')}
        </Title>
        <Text type="secondary" className="instance-resources-brand">
          {t('qxmods.brand')}
        </Text>
      </div>
      <Paragraph type="secondary" className="qxmods-catalog-intro">
        {t('qxmods.catalogIntro')}
      </Paragraph>
      <Tabs activeKey={activeTab} onChange={(key) => setActiveTab(key as ModProjectType)} items={tabItems} />

      <Drawer
        title={
          detailItem ? (
            <Space wrap>
              <span>{detailItem.name}</span>
              <ModSourceBadge source={detailItem.source} />
            </Space>
          ) : (
            t('qxmods.detailTitle')
          )
        }
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={480}
        extra={
          detailItem ? (
            <a href={detailItem.external_url} target="_blank" rel="noreferrer">
              <LinkOutlined /> {t('qxmods.viewOnSource')}
            </a>
          ) : null
        }
      >
        {detailItem ? (
          <div className="qxmods-detail">
            <Paragraph type="secondary">{detailItem.summary}</Paragraph>
            <Paragraph className="qxmods-detail-attribution">{t('qxmods.detailAttribution')}</Paragraph>
            {versionsLoading ? (
              <Spin />
            ) : versions.length === 0 ? (
              <Empty description={t('qxmods.noVersions')} />
            ) : (
              <>
                <Text strong>{t('qxmods.selectVersion')}</Text>
                <List
                  size="small"
                  className="qxmods-version-list"
                  dataSource={versions}
                  renderItem={(version) => (
                    <List.Item
                      key={version.id}
                      className={
                        selectedVersionId === version.id ? 'qxmods-version-item--selected' : ''
                      }
                      onClick={() => setSelectedVersionId(version.id)}
                    >
                      <div>
                        <Text>{version.version_number}</Text>
                        {version.game_versions?.length ? (
                          <Text type="secondary" className="qxmods-version-meta">
                            {' '}
                            · MC {version.game_versions.join(', ')}
                          </Text>
                        ) : null}
                      </div>
                    </List.Item>
                  )}
                />
              </>
            )}
            {showSyncButton ? (
              <Button
                type="primary"
                icon={<CloudSyncOutlined />}
                className="qxmods-sync-btn"
                onClick={handleSyncClick}
              >
                {t('qxmods.sync.action')}
              </Button>
            ) : null}
          </div>
        ) : null}
      </Drawer>

      <ModSyncModal
        open={syncOpen}
        selection={syncSelection}
        instanceLoader={instance.loader}
        onClose={() => setSyncOpen(false)}
      />
    </section>
  );
}
