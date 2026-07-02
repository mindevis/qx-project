import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Modal, Spin, Typography } from 'antd';
import { AppstoreOutlined, DeleteOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { api, type InstanceResource, type ModProjectType } from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ResourceMetaBadges } from '@/components/ResourceMetaBadges';
import { InstanceServerSyncPanel } from '@/components/InstanceServerSyncPanel';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { launcherCatalogTabs } from '@/lib/launcherInstanceCapabilities';
import './InstanceResourcesPanel.css';

const { Text, Title } = Typography;

export function InstanceInstalledResources() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, basePath, canSync } = useInstanceMods();
  const [items, setItems] = useState<InstanceResource[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [removingKey, setRemovingKey] = useState<string>();

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

  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      const res = await api.listInstanceResources(instance.id);
      setItems(res.items ?? []);
      message.success(t('launcherInstanceResources.refreshed'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.installed.loadFailed'));
    } finally {
      setRefreshing(false);
    }
  }, [instance.id, message, t]);

  useEffect(() => {
    void load();
  }, [load]);

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

  return (
    <section className="instance-resources-panel instance-resources-panel--standalone" aria-label={t('qxmods.sectionTitle')}>
      <div className="instance-resources-header">
        <Title level={4} className="instance-resources-title">
          {t('launcherInstanceResources.installedTitle')}
        </Title>
        <Text type="secondary" className="instance-resources-brand">
          {t('qxmods.brand')}
        </Text>
      </div>

      {!loading && items.length > 0 ? (
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
      ) : null}

      {canSync ? <InstanceServerSyncPanel items={items} /> : null}

      <div className="launcher-resources-toolbar">
        <Text type="secondary">{t('qxmods.installed.intro')}</Text>
        <div className="launcher-resources-toolbar-actions">
          <Button
            icon={<ReloadOutlined spin={refreshing} />}
            loading={refreshing}
            onClick={() => void refresh()}
          >
            {t('launcherInstanceResources.refresh')}
          </Button>
          <Link to={`${basePath}/catalog`}>
            <Button type="primary" icon={<PlusOutlined />}>
              {t('qxmods.installed.add')}
            </Button>
          </Link>
        </div>
      </div>

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
            <ul className="launcher-resources-grid">
              {sectionItems.map((item) => (
                <li key={`${item.source}:${item.project_id ?? item.filename}`}>
                  <article className="launcher-resource-card">
                    {item.icon_url ? (
                      <img src={item.icon_url} alt="" className="launcher-resource-card-icon" />
                    ) : (
                      <span
                        className="launcher-resource-card-icon launcher-resource-card-icon--placeholder"
                        aria-hidden
                      />
                    )}
                    <div className="launcher-resource-card-body">
                      <div className="launcher-resource-card-title">
                        <span>{item.project_name}</span>
                        <ModSourceBadge source={item.source} />
                      </div>
                      <ResourceMetaBadges item={item} t={t} />
                    </div>
                    <Button
                      type="text"
                      danger
                      size="small"
                      className="launcher-resource-card-remove"
                      icon={<DeleteOutlined />}
                      loading={removingKey === resourceKey(item)}
                      aria-label={t('qxmods.uninstall.action')}
                      onClick={() => handleRemove(item)}
                    />
                  </article>
                </li>
              ))}
            </ul>
          </div>
        ))
      )}
    </section>
  );
}
