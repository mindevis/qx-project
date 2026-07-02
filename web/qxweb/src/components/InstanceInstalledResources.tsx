import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Modal, Select, Spin, Typography, Upload } from 'antd';
import { AppstoreOutlined, DeleteOutlined, UnorderedListOutlined, UploadOutlined } from '@ant-design/icons';
import { api, type InstanceResource, type ModProjectType, type ModSyncSide } from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { ResourceMetaBadges } from '@/components/ResourceMetaBadges';
import {
  InstanceResourceSyncButton,
  InstanceServerSyncProvider,
} from '@/components/InstanceServerSyncPanel';
import { SegmentedControl } from '@/components/SegmentedControl';
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
  onSideChange: (item: InstanceResource, side: ModSyncSide | '') => void;
  sideSavingKey?: string;
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
  t,
}: InstalledResourceItemProps) {
  const removeButton = (
    <Button
      type="text"
      danger
      size="small"
      className="launcher-resource-card-remove"
      icon={<DeleteOutlined />}
      loading={removingKey === resourceKey(item)}
      aria-label={t('qxmods.uninstall.action')}
      onClick={() => onRemove(item)}
    />
  );

  const sideSelect =
    item.resource_type === 'mod' ||
    item.resource_type === 'resourcepack' ||
    item.resource_type === 'shader' ? (
      <Select
        size="small"
        className="launcher-resource-side-select"
        loading={sideSavingKey === resourceKey(item)}
        value={item.side_override ?? ''}
        options={sideOptions(t)}
        aria-label={t('qxmods.side.editAria')}
        onChange={(value) => onSideChange(item, (value || '') as ModSyncSide | '')}
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
            <span>{item.project_name}</span>
            <ModSourceBadge source={item.source} />
          </div>
          <ResourceMetaBadges item={item} t={t} />
          {sideSelect}
        </div>
        <div className="qxmods-installed-item-actions">
          <InstanceResourceSyncButton item={item} />
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
          <span>{item.project_name}</span>
          <ModSourceBadge source={item.source} />
        </div>
        <ResourceMetaBadges item={item} t={t} />
        {sideSelect}
      </div>
      <div className="launcher-resource-card-actions">
        <InstanceResourceSyncButton item={item} />
        {removeButton}
      </div>
    </article>
  );
}

export function InstanceInstalledResources() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, basePath, canSync } = useInstanceMods();
  const { viewMode, setViewMode } = useInstalledResourcesViewMode();
  const [items, setItems] = useState<InstanceResource[]>([]);
  const [resolvedIconUrls, setResolvedIconUrls] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [removingKey, setRemovingKey] = useState<string>();
  const [sideSavingKey, setSideSavingKey] = useState<string>();
  const [uploading, setUploading] = useState(false);

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

  const grouped = useMemo(() => {
    const map = new Map<ModProjectType, InstanceResource[]>();
    for (const item of items) {
      const list = map.get(item.resource_type) ?? [];
      list.push(item);
      map.set(item.resource_type, list);
    }
    for (const list of map.values()) {
      list.sort((a, b) => a.project_name.localeCompare(b.project_name, undefined, { sensitivity: 'base' }));
    }
    return map;
  }, [items]);

  const sections = useMemo(
    () =>
      typeOrder
        .map((type) => ({ type, items: grouped.get(type) ?? [] }))
        .filter((section) => section.items.length > 0),
    [grouped, typeOrder],
  );

  const stats = useMemo(() => {
    const counts = new Map<ModProjectType, number>();
    for (const item of items) {
      counts.set(item.resource_type, (counts.get(item.resource_type) ?? 0) + 1);
    }
    return counts;
  }, [items]);

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

  const handleSideChange = async (item: InstanceResource, side: ModSyncSide | '') => {
    const key = resourceKey(item);
    setSideSavingKey(key);
    try {
      await api.patchInstanceResource(instance.id, {
        source: item.source,
        project_id: item.project_id,
        filename: item.filename,
        resource_type: item.resource_type,
        side_override: side,
      });
      setItems((prev) =>
        prev.map((entry) =>
          resourceKey(entry) === key ? { ...entry, side_override: side || undefined } : entry,
        ),
      );
      message.success(t('qxmods.side.saved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.side.saveFailed'));
    } finally {
      setSideSavingKey(undefined);
    }
  };

  const handleRemove = (item: InstanceResource) => {
    Modal.confirm({
      title: t('qxmods.uninstall.confirmTitle'),
      content: t('qxmods.uninstall.confirmBody', { name: item.project_name }),
      okText: t('qxmods.uninstall.action'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
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
          throw e;
        } finally {
          setRemovingKey(undefined);
        }
      },
    });
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
        <div className="launcher-resources-toolbar">
          <div className="launcher-resources-stats" aria-label={t('launcherInstanceResources.statsAria')}>
            <span className="launcher-resources-stat">
              {t('launcherInstanceResources.statTotal')}: <strong>{items.length}</strong>
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
          <SegmentedControl
            iconOnly
            value={viewMode}
            onChange={setViewMode}
            groupLabel={t('launcherInstanceResources.viewModeAria')}
            options={[
              {
                value: 'list',
                label: <UnorderedListOutlined aria-hidden />,
                ariaLabel: t('launcherInstanceResources.viewList'),
              },
              {
                value: 'cards',
                label: <AppstoreOutlined aria-hidden />,
                ariaLabel: t('launcherInstanceResources.viewCards'),
              },
            ]}
          />
        </div>
      ) : null}

      {loading ? (
        <div className="qxmods-loading">
          <Spin />
        </div>
      ) : items.length === 0 ? (
        <div className="launcher-resources-empty">
          <AppstoreOutlined className="launcher-resources-empty-icon" aria-hidden />
          <Title level={5}>{t('qxmods.installed.empty')}</Title>
          <Text type="secondary">{t('launcherInstanceResources.emptyHint')}</Text>
          <Link to={`${basePath}/catalog`}>
            <Button type="primary" icon={<AppstoreOutlined />}>
              {t('launcherInstanceResources.browseCatalog')}
            </Button>
          </Link>
        </div>
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
                    onSideChange={(entry, side) => void handleSideChange(entry, side)}
                    sideSavingKey={sideSavingKey}
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
