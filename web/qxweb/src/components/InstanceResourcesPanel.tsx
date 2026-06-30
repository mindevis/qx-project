import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Drawer,
  Empty,
  Input,
  List,
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

export function InstanceResourcesPanel({
  instance,
  canSync,
  layout = 'embedded',
}: InstanceResourcesPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [activeTab, setActiveTab] = useState<ModProjectType>('mod');
  const [query, setQuery] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [items, setItems] = useState<ModCatalogItem[]>([]);
  const [curseforgeEnabled, setCurseforgeEnabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailItem, setDetailItem] = useState<ModCatalogItem | null>(null);
  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [selectedVersionId, setSelectedVersionId] = useState<string>();
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncSelection, setSyncSelection] = useState<ModSyncSelection | null>(null);

  const modded = isModdedLauncherLoader(instance.loader);

  const runSearch = useCallback(async () => {
    const q = searchInput.trim();
    if (!q) {
      setItems([]);
      return;
    }
    setQuery(q);
    setLoading(true);
    try {
      const res = await api.searchMods({
        q,
        type: activeTab,
        loader: instance.loader,
        mc_version: instance.mc_version,
      });
      setItems(res.items ?? []);
      setCurseforgeEnabled(res.curseforge_enabled ?? false);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.searchFailed'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [activeTab, instance.loader, instance.mc_version, message, searchInput, t]);

  useEffect(() => {
    setItems([]);
    setQuery('');
  }, [activeTab]);

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

  if (!modded) {
    return null;
  }

  const tabItems = TAB_TYPES.map((type) => ({
    key: type,
    label: t(`qxmods.tabs.${type}`),
    children: (
      <div className="qxmods-tab-panel">
        <div className="qxmods-search-bar">
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder={t('qxmods.searchPlaceholder')}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onPressEnter={() => void runSearch()}
          />
          <Button type="primary" onClick={() => void runSearch()} loading={loading}>
            {t('qxmods.search')}
          </Button>
        </div>
        <Paragraph type="secondary" className="qxmods-attribution">
          {t('qxmods.attribution')}
          {!curseforgeEnabled ? ` ${t('qxmods.curseforgeDisabled')}` : ''}
        </Paragraph>
        {loading ? (
          <div className="qxmods-loading">
            <Spin />
          </div>
        ) : query && items.length === 0 ? (
          <Empty description={t('qxmods.empty')} />
        ) : (
          <List
            className="qxmods-results"
            dataSource={items}
            locale={{ emptyText: t('qxmods.searchPrompt') }}
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
        )}
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
