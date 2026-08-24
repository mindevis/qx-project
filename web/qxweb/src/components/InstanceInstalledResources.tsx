import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Empty, Input, Popconfirm, Segmented, Select, Spin, Tag, Typography, Upload } from 'antd';
import { AppstoreOutlined, DeleteOutlined, SearchOutlined, UnorderedListOutlined, UploadOutlined } from '@ant-design/icons';
import {
  api,
  ApiRequestError,
  type InstanceResource,
  type ModProjectType,
  type ModSyncSide,
} from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { ResourceMetaBadges } from '@/components/ResourceMetaBadges';
import {
  InstanceResourceSyncButton,
  InstanceServerSyncProvider,
  useInstanceServerSync,
} from '@/components/InstanceServerSyncPanel';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { launcherCatalogTabs } from '@/lib/launcherInstanceCapabilities';
import { fetchMissingResourceIcons, instanceResourceIconKey } from '@/lib/instanceResourceIcons';
import {
  type InstalledResourcesViewMode,
  useInstalledResourcesViewMode,
} from '@/lib/installedResourcesView';
import './InstanceResourcesPanel.css';
import '../pages/LauncherInstanceResourcesPage.css';

const { Text, Title } = Typography;

type InstalledResourceItemProps = {
  item: InstanceResource;
  viewMode: InstalledResourcesViewMode;
  iconUrl?: string;
  removingKey?: string;
  resourceKey: (item: InstanceResource) => string;
  onRemove: (item: InstanceResource) => void;
  onSideChange: (item: InstanceResource, side: ModSyncSide | '') => Promise<boolean>;
  sideSavingKey?: string;
  basePath: string;
  contentLocked?: boolean;
  t: ReturnType<typeof useI18n>['t'];
};

const sideOptions = (t: ReturnType<typeof useI18n>['t']) => [
  { value: '', label: t('qxmods.side.auto') },
  { value: 'client', label: t('qxmods.side.client') },
  { value: 'server', label: t('qxmods.side.server') },
  { value: 'both', label: t('qxmods.side.both') },
];

function InstalledResourceItem({
  item,
  viewMode,
  iconUrl,
  removingKey,
  resourceKey,
  onRemove,
  onSideChange,
  sideSavingKey,
  basePath,
  contentLocked,
  t,
}: InstalledResourceItemProps) {
  const { offerRemoveFromServerMods } = useInstanceServerSync();
  const removeButton = contentLocked ? null : (
    <Popconfirm
      title={t('qxmods.uninstall.confirmTitle')}
      description={t('qxmods.uninstall.confirmBody', { name: item.project_name })}
      okText={t('qxmods.uninstall.action')}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true }}
      onConfirm={() => onRemove(item)}
    >
      <Button
        type="text"
        danger
        size="small"
        className="launcher-resource-card-remove"
        icon={<DeleteOutlined />}
        loading={removingKey === resourceKey(item)}
        aria-label={t('qxmods.uninstall.action')}
      />
    </Popconfirm>
  );

  const sideSelect =
    !contentLocked &&
    (item.resource_type === 'mod' ||
    item.resource_type === 'resourcepack' ||
    item.resource_type === 'shader') ? (
      <Select
        size="small"
        className="launcher-resource-side-select"
        loading={sideSavingKey === resourceKey(item)}
        value={item.side_override ?? ''}
        options={sideOptions(t)}
        aria-label={t('qxmods.side.editAria')}
        onChange={(value) => {
          const side = (value || '') as ModSyncSide | '';
          void (async () => {
            const ok = await onSideChange(item, side);
            if (ok && side === 'client') {
              offerRemoveFromServerMods(item);
            }
          })();
        }}
      />
    ) : null;

  if (viewMode === 'list') {
    return (
      <article className="qxmods-installed-item launcher-resources-list-item">
        <ModCatalogIcon
          url={iconUrl}
          name={item.project_name}
          size={40}
          className="qxmods-installed-icon"
        />
        <div className="qxmods-installed-item-content">
          <div className="qxmods-installed-item-title">
            <span className="qxmods-title-with-source">
              {item.project_id && item.source !== 'upload' ? (
                <Link
                  to={`${basePath}/catalog/${item.source}/${item.project_id}`}
                  className="launcher-resource-item-link"
                >
                  {item.project_name}
                </Link>
              ) : (
                <span>{item.project_name}</span>
              )}
              <ModSourceBadge source={item.source} />
            </span>
          </div>
          <ResourceMetaBadges item={item} />
          {sideSelect}
        </div>
        <div className="qxmods-installed-item-actions">
          <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--type launcher-resource-type-badge">
            {t(`qxmods.tabs.${item.resource_type}`)}
          </Tag>
          {contentLocked ? null : <InstanceResourceSyncButton item={item} />}
          {removeButton}
        </div>
      </article>
    );
  }

  return (
    <article className="launcher-resource-card">
      <ModCatalogIcon
        url={iconUrl}
        name={item.project_name}
        size={48}
        className="launcher-resource-card-icon"
      />
      <div className="launcher-resource-card-body">
        <div className="launcher-resource-card-title">
          <span className="qxmods-title-with-source">
            {item.project_id && item.source !== 'upload' ? (
              <Link
                to={`${basePath}/catalog/${item.source}/${item.project_id}`}
                className="launcher-resource-item-link"
              >
                {item.project_name}
              </Link>
            ) : (
              <span>{item.project_name}</span>
            )}
            <ModSourceBadge source={item.source} />
          </span>
        </div>
        <ResourceMetaBadges item={item} />
        {sideSelect}
      </div>
      <div className="launcher-resource-card-actions">
        <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--type launcher-resource-type-badge">
          {t(`qxmods.tabs.${item.resource_type}`)}
        </Tag>
        {contentLocked ? null : <InstanceResourceSyncButton item={item} />}
        {removeButton}
      </div>
    </article>
  );
}

export function InstanceInstalledResources() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, basePath, canSync, contentLocked } = useInstanceMods();
  const { viewMode, setViewMode } = useInstalledResourcesViewMode();
  const [items, setItems] = useState<InstanceResource[]>([]);
  const [resolvedIconUrls, setResolvedIconUrls] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [removingKey, setRemovingKey] = useState<string>();
  const [sideSavingKey, setSideSavingKey] = useState<string>();
  const [uploading, setUploading] = useState(false);
  const [searchInput, setSearchInput] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listInstanceResources(instance.id);
      setItems(res.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.installed.loadFailed'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [instance.id, message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      const icons = await fetchMissingResourceIcons(items);
      if (!cancelled) {
        setResolvedIconUrls(icons);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [items]);

  const typeOrder = useMemo(() => launcherCatalogTabs(instance.loader), [instance.loader]);

  const filteredItems = useMemo(() => {
    const query = appliedSearch.trim().toLowerCase();
    if (!query) return items;
    return items.filter((item) => {
      const name = item.project_name.toLowerCase();
      const filename = item.filename.toLowerCase();
      const projectId = item.project_id?.toLowerCase() ?? '';
      return name.includes(query) || filename.includes(query) || projectId.includes(query);
    });
  }, [appliedSearch, items]);

  const grouped = useMemo(() => {
    const map = new Map<ModProjectType, InstanceResource[]>();
    for (const item of filteredItems) {
      const list = map.get(item.resource_type) ?? [];
      list.push(item);
      map.set(item.resource_type, list);
    }
    for (const list of map.values()) {
      list.sort((a, b) => a.project_name.localeCompare(b.project_name, undefined, { sensitivity: 'base' }));
    }
    return map;
  }, [filteredItems]);

  const sections = useMemo(
    () =>
      typeOrder
        .map((type) => ({ type, items: grouped.get(type) ?? [] }))
        .filter((section) => section.items.length > 0),
    [grouped, typeOrder],
  );

  const stats = useMemo(() => {
    const counts = new Map<ModProjectType, number>();
    for (const item of filteredItems) {
      counts.set(item.resource_type, (counts.get(item.resource_type) ?? 0) + 1);
    }
    return counts;
  }, [filteredItems]);

  const resourceKey = (item: InstanceResource) =>
    `${item.source}:${item.project_id ?? item.filename}:${item.resource_type}`;

  const handleUpload = async (file: File) => {
    if (!canSync) {
      message.warning(t('launcher.instanceSettingsModsConfigNote'));
      return false;
    }
    setUploading(true);
    try {
      await api.uploadInstanceResource(instance.id, file);
      message.success(t('qxmods.upload.completed'));
      await load();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.upload.failed'));
    } finally {
      setUploading(false);
    }
    return false;
  };

  const handleSideChange = async (
    item: InstanceResource,
    side: ModSyncSide | '',
  ): Promise<boolean> => {
    const key = resourceKey(item);
    const previousSide = item.side_override;
    setSideSavingKey(key);
    // Reflect the choice immediately, then roll back if the save fails so the
    // dropdown never silently snaps back to the old value.
    setItems((prev) =>
      prev.map((entry) =>
        resourceKey(entry) === key ? { ...entry, side_override: side || undefined } : entry,
      ),
    );
    try {
      await api.patchInstanceResource(instance.id, {
        source: item.source,
        project_id: item.project_id,
        filename: item.filename,
        resource_type: item.resource_type,
        side_override: side,
      });
      message.success(t('qxmods.side.saved'));
      return true;
    } catch (e) {
      setItems((prev) =>
        prev.map((entry) =>
          resourceKey(entry) === key ? { ...entry, side_override: previousSide } : entry,
        ),
      );
      if (e instanceof ApiRequestError && e.apiCode === 'NOT_FOUND') {
        message.error(t('qxmods.side.notRegistered'));
      } else {
        message.error(e instanceof Error ? e.message : t('qxmods.side.saveFailed'));
      }
      return false;
    } finally {
      setSideSavingKey(undefined);
    }
  };

  const handleRemove = async (item: InstanceResource) => {
    const key = resourceKey(item);
    setRemovingKey(key);
    try {
      await api.deleteInstanceResource(instance.id, {
        source: item.source,
        project_id: item.project_id,
        filename: item.filename,
        resource_type: item.resource_type,
      });
      setItems((prev) => prev.filter((entry) => resourceKey(entry) !== key));
      message.success(t('qxmods.uninstall.completed'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.uninstall.failed'));
    } finally {
      setRemovingKey(undefined);
    }
  };

  const listClassName = viewMode === 'list' ? 'qxmods-installed-list' : 'launcher-resources-grid';

  return (
    <InstanceServerSyncProvider items={items}>
      <section className="instance-resources-panel instance-resources-panel--standalone" aria-label={t('qxmods.sectionTitle')}>
      <div className="instance-resources-header">
        <Title level={4} className="instance-resources-title">
          {t('launcherInstanceResources.installedTitle')}
        </Title>
        <Text type="secondary" className="instance-resources-brand">
          {t('qxmods.brand')}
        </Text>
      </div>

      {canSync ? (
        <Upload.Dragger
          className="instance-resources-upload"
          accept=".jar,.zip,.mrpack"
          showUploadList={false}
          disabled={uploading}
          beforeUpload={(file) => {
            void handleUpload(file);
            return false;
          }}
        >
          <p className="ant-upload-drag-icon">
            <UploadOutlined />
          </p>
          <p className="ant-upload-text">{t('qxmods.upload.dropHint')}</p>
          <p className="ant-upload-hint">{t('qxmods.upload.extensionsHint')}</p>
        </Upload.Dragger>
      ) : null}

      {!loading && items.length > 0 ? (
        <>
          <div className="launcher-resources-search">
            <Input.Search
              allowClear
              enterButton={appliedSearch ? t('qxmods.applySearch') : t('qxmods.search')}
              prefix={<SearchOutlined aria-hidden />}
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
          </div>
          <div className="launcher-resources-toolbar">
            <div className="launcher-resources-stats" aria-label={t('launcherInstanceResources.statsAria')}>
              <span className="launcher-resources-stat">
                {t('launcherInstanceResources.statTotal')}: <strong>{filteredItems.length}</strong>
              </span>
              {typeOrder.map((type) => {
                const count = stats.get(type) ?? 0;
                if (count === 0) return null;
                return (
                  <span key={type} className="launcher-resources-stat">
                    {t(`qxmods.tabs.${type}`)}: <strong>{count}</strong>
                  </span>
                );
              })}
            </div>
            <Segmented
              size="small"
              value={viewMode}
              aria-label={t('launcherInstanceResources.viewModeAria')}
              onChange={(value) => setViewMode(value as InstalledResourcesViewMode)}
              options={[
                {
                  value: 'list',
                  label: (
                    <span aria-label={t('launcherInstanceResources.viewList')}>
                      <UnorderedListOutlined aria-hidden />
                    </span>
                  ),
                },
                {
                  value: 'cards',
                  label: (
                    <span aria-label={t('launcherInstanceResources.viewCards')}>
                      <AppstoreOutlined aria-hidden />
                    </span>
                  ),
                },
              ]}
            />
          </div>
        </>
      ) : null}

      {loading ? (
        <div className="qxmods-loading">
          <Spin />
        </div>
      ) : items.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <div>
              <Title level={5}>{t('qxmods.installed.empty')}</Title>
              <Text type="secondary">
                {t(
                  contentLocked
                    ? 'launcherInstanceResources.emptyHintLocked'
                    : 'launcherInstanceResources.emptyHint',
                )}
              </Text>
            </div>
          }
        >
          {contentLocked ? null : (
            <Link to={`${basePath}/catalog`}>
              <Button type="primary" icon={<AppstoreOutlined />}>
                {t('launcherInstanceResources.browseCatalog')}
              </Button>
            </Link>
          )}
        </Empty>
      ) : filteredItems.length === 0 ? (
        <Empty description={t('qxmods.empty')} />
      ) : (
        sections.map(({ type, items: sectionItems }) => (
          <div key={type} className="launcher-resources-section">
            <Title level={5} className="launcher-resources-section-title">
              {t(`qxmods.tabs.${type}`)}
              <span className="launcher-resources-section-count">{sectionItems.length}</span>
            </Title>
            <ul className={listClassName}>
              {sectionItems.map((item) => {
                const iconKey = instanceResourceIconKey(item);
                const iconUrl = item.icon_url ?? (iconKey ? resolvedIconUrls[iconKey] : undefined);
                return (
                <li key={`${item.source}:${item.project_id ?? item.filename}`}>
                  <InstalledResourceItem
                    item={item}
                    viewMode={viewMode}
                    iconUrl={iconUrl}
                    removingKey={removingKey}
                    resourceKey={resourceKey}
                    onRemove={handleRemove}
                    onSideChange={handleSideChange}
                    sideSavingKey={sideSavingKey}
                    basePath={basePath}
                    contentLocked={contentLocked}
                    t={t}
                  />
                </li>
                );
              })}
            </ul>
          </div>
        ))
      )}
      </section>
    </InstanceServerSyncProvider>
  );
}
