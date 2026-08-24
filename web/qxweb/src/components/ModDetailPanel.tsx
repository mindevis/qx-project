import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { Button, Empty, Select, Spin, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, CheckCircleOutlined, CloudSyncOutlined } from '@ant-design/icons';
import {
  api,
  type ModCatalogItem,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { CatalogSourceLinks } from '@/components/CatalogSourceSwitch';
import { ModCatalogIcon } from '@/components/ModCatalogIcon';
import { ModCatalogInstallControls } from '@/components/ModCatalogInstallControls';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { modSupportsServerSync } from '@/lib/modSync';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { catalogLoaderForType } from '@/lib/launcherInstanceCapabilities';
import { cachedGetModProject } from '@/lib/modCatalogCache';
import './InstanceResourcesPanel.css';

const { Text, Paragraph, Title } = Typography;

export function ModDetailPanel() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, canSync, basePath } = useInstanceMods();
  const location = useLocation();
  const catalogSiblings = (location.state as { catalogSiblings?: ModCatalogItem[] } | null)?.catalogSiblings;
  const { source, projectId } = useParams<{ source: ModSource; projectId: string }>();
  const [detail, setDetail] = useState<ModCatalogItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [mcVersion, setMcVersion] = useState(instance.mc_version);
  const [loader, setLoader] = useState(instance.loader);
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncSelection, setSyncSelection] = useState<ModSyncSelection | null>(null);
  const [lastInstalledVersion, setLastInstalledVersion] = useState<ModVersion | null>(null);
  const [installedProjectIds, setInstalledProjectIds] = useState<Set<string>>(new Set());
  const [gameVersions, setGameVersions] = useState<string[]>([]);
  const [loaders, setLoaders] = useState<string[]>([]);

  const resourceType: ModProjectType = detail?.project_type ?? 'mod';
  const catalogLoader = catalogLoaderForType(instance.loader, resourceType);
  const isInstalled =
    source != null && detail != null && installedProjectIds.has(`${source}:${detail.id}`);

  const refreshInstalled = useCallback(async () => {
    try {
      const res = await api.listInstanceResources(instance.id);
      setInstalledProjectIds(
        new Set(
          (res.items ?? [])
            .filter((r) => r.project_id)
            .map((r) => `${r.source}:${r.project_id}`),
        ),
      );
    } catch {
      setInstalledProjectIds(new Set());
    }
  }, [instance.id]);

  const handleInstalled = useCallback(
    (version: ModVersion) => {
      setLastInstalledVersion(version);
      void refreshInstalled();
      if (
        canSync &&
        detail &&
        source &&
        modSupportsServerSync(detail) &&
        resourceType === 'mod'
      ) {
        setSyncSelection({
          source,
          projectId: detail.id,
          projectName: detail.name,
          version,
        });
        setSyncOpen(true);
      }
    },
    [canSync, detail, refreshInstalled, resourceType, source],
  );

  useEffect(() => {
    if (!source || !projectId) return;
    let cancelled = false;
    setLoading(true);
    void (async () => {
      try {
        const project = await cachedGetModProject(source, projectId);
        if (cancelled) return;
        setDetail(project);
        setGameVersions(project.game_versions ?? []);
        setLoaders(project.loaders ?? []);
      } catch (e) {
        if (cancelled) return;
        message.error(formatModCatalogError(e, t, 'qxmods.browseFailed'));
        setDetail(null);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [message, projectId, source, t]);

  useEffect(() => {
    void refreshInstalled();
  }, [refreshInstalled, lastInstalledVersion]);

  const mcVersionOptions = useMemo(() => {
    const unique = [...new Set([instance.mc_version, ...gameVersions])];
    return unique.map((value) => ({ value, label: value }));
  }, [gameVersions, instance.mc_version]);

  const loaderOptions = useMemo(() => {
    const unique = [...new Set([instance.loader, ...loaders])];
    return unique.map((value) => ({ value, label: value }));
  }, [instance.loader, loaders]);

  const showSyncButton =
    canSync &&
    detail != null &&
    lastInstalledVersion != null &&
    modSupportsServerSync(detail) &&
    resourceType === 'mod';

  const handleSyncClick = () => {
    if (!detail || !lastInstalledVersion || !source) return;
    setSyncSelection({
      source,
      projectId: detail.id,
      projectName: detail.name,
      version: lastInstalledVersion,
    });
    setSyncOpen(true);
  };

  if (loading) {
    return (
      <div className="qxmods-loading">
        <Spin />
      </div>
    );
  }

  if (!detail || !source) {
    return <Empty description={t('qxmods.detail.notFound')} />;
  }

  return (
    <section className="instance-resources-panel instance-resources-panel--standalone qxmods-detail-page">
      <div className="qxmods-page-toolbar">
        <Link to={`${basePath}/catalog`} className="launcher-instance-detail-back">
          <ArrowLeftOutlined /> {t('qxmods.detail.backToCatalog')}
        </Link>
        <CatalogSourceLinks items={catalogSiblings && catalogSiblings.length > 1 ? catalogSiblings : detail ? [detail] : []} />
      </div>
      <div className="qxmods-detail-header">
        <ModCatalogIcon url={detail.icon_url} name={detail.name} size={72} className="qxmods-detail-icon" />
        <div className="qxmods-detail-header-body">
          <Title level={3} className="qxmods-detail-title">
            {detail.name} <ModSourceBadge source={detail.source} />
            {isInstalled ? (
              <Tag icon={<CheckCircleOutlined />} color="success" className="qxmods-installed-badge">
                {t('qxmods.installed.badge')}
              </Tag>
            ) : null}
          </Title>
          <Paragraph type="secondary">{detail.summary}</Paragraph>
        </div>
      </div>
      <Paragraph className="qxmods-detail-attribution">{t('qxmods.detailAttribution')}</Paragraph>
      <div className="qxmods-detail-filters">
        <label className="qxmods-filter-field">
          <Text type="secondary" className="qxmods-filter-label">
            {t('qxmods.detail.mcVersion')}
          </Text>
          <Select
            value={mcVersion}
            options={mcVersionOptions}
            onChange={setMcVersion}
            className="qxmods-filter-select"
          />
        </label>
        {resourceType !== 'datapack' ? (
          <label className="qxmods-filter-field">
            <Text type="secondary" className="qxmods-filter-label">
              {t('qxmods.detail.loader')}
            </Text>
            <Select
              value={loader}
              options={loaderOptions}
              onChange={setLoader}
              className="qxmods-filter-select"
            />
          </label>
        ) : null}
      </div>
      <div className="qxmods-detail-install-bar">
        <Text strong className="qxmods-detail-install-label">
          {t('qxmods.selectVersion')}
        </Text>
        <ModCatalogInstallControls
          source={source}
          projectId={detail.id}
          projectName={detail.name}
          projectType={resourceType}
          iconUrl={detail.icon_url}
          downloads={detail.downloads}
          clientSide={detail.client_side}
          serverSide={detail.server_side}
          loader={catalogLoaderForType(loader, resourceType) ?? catalogLoader}
          mcVersion={mcVersion}
          installedProjectIds={installedProjectIds}
          layout="inline"
          eagerVersions
          showDependencies
          selectClassName="qxmods-install-version-select--detail"
          onInstalled={handleInstalled}
          onUninstalled={() => {
            setLastInstalledVersion(null);
            void refreshInstalled();
          }}
          onDependencyInstalled={() => void refreshInstalled()}
        />
      </div>
      {showSyncButton ? (
        <Button
          type="default"
          icon={<CloudSyncOutlined />}
          className="qxmods-sync-btn"
          onClick={handleSyncClick}
        >
          {t('qxmods.sync.withServer')}
        </Button>
      ) : null}
      <ModSyncModal
        open={syncOpen}
        selection={syncSelection}
        instanceLoader={instance.loader}
        onClose={() => setSyncOpen(false)}
      />
    </section>
  );
}
