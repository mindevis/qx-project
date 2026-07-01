import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Empty, Spin, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { api, type InstanceResource } from '@/api/client';
import { InstanceServerBinding } from '@/components/InstanceServerBinding';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { formatDownloadCount, formatFileSize } from '@/lib/formatFileSize';
import './InstanceResourcesPanel.css';

const { Text, Title } = Typography;

export function InstanceInstalledResources({ layout = 'standalone' }: { layout?: 'embedded' | 'standalone' }) {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, basePath } = useInstanceMods();
  const [items, setItems] = useState<InstanceResource[]>([]);
  const [loading, setLoading] = useState(true);

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
      <InstanceServerBinding />
      <div className="qxmods-installed-toolbar">
        <Text type="secondary">{t('qxmods.installed.intro')}</Text>
        <Link to={`${basePath}/catalog`}>
          <Button type="primary" icon={<PlusOutlined />}>
            {t('qxmods.installed.add')}
          </Button>
        </Link>
      </div>
      {loading ? (
        <div className="qxmods-loading">
          <Spin />
        </div>
      ) : items.length === 0 ? (
        <Empty description={t('qxmods.installed.empty')}>
          <Link to={`${basePath}/catalog`}>
            <Button type="primary">{t('qxmods.installed.add')}</Button>
          </Link>
        </Empty>
      ) : (
        <ul className="qxmods-installed-list">
          {items.map((item) => (
            <li key={`${item.source}:${item.project_id ?? item.filename}`} className="qxmods-installed-item">
              {item.icon_url ? (
                <img src={item.icon_url} alt="" className="qxmods-installed-icon" />
              ) : (
                <span className="qxmods-installed-icon qxmods-installed-icon--placeholder" />
              )}
              <div className="qxmods-installed-item-content">
                <div className="qxmods-installed-item-title">
                  {item.project_name} <ModSourceBadge source={item.source} />
                </div>
                <Text type="secondary">
                  {t(`qxmods.tabs.${item.resource_type}`)}
                  {item.version_number ? ` · ${item.version_number}` : ''}
                  {item.filename ? ` · ${item.filename}` : ''}
                  {item.file_size ? ` · ${formatFileSize(item.file_size)}` : ''}
                  {item.downloads ? ` · ${formatDownloadCount(item.downloads)} ${t('qxmods.installed.downloads')}` : ''}
                </Text>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
