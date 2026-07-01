import { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Button, Empty, Select, Space, Spin, Typography } from 'antd';
import { ArrowLeftOutlined, CloudSyncOutlined, LinkOutlined } from '@ant-design/icons';
import {
  api,
  type ModCatalogItem,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { ModInstallDepsModal, type InstallItem } from '@/components/ModInstallDepsModal';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { useModInstall } from '@/hooks/useModInstall';
import { modSupportsServerSync } from '@/lib/modSync';
import { formatModCatalogError } from '@/lib/modCatalogError';
import './InstanceResourcesPanel.css';

const { Text, Paragraph, Title } = Typography;

export function ModDetailPanel() {
  const { t } = useI18n();
  const message = useMessage();
  const { instance, canSync, basePath } = useInstanceMods();
  const { source, projectId } = useParams<{ source: ModSource; projectId: string }>();
  const [detail, setDetail] = useState<ModCatalogItem | null>(null);
  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [mcVersion, setMcVersion] = useState(instance.mc_version);
  const [loader, setLoader] = useState(instance.loader);
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncSelection, setSyncSelection] = useState<ModSyncSelection | null>(null);
  const [lastInstalledVersion, setLastInstalledVersion] = useState<ModVersion | null>(null);
  const [depsOpen, setDepsOpen] = useState(false);
  const [pendingVersion, setPendingVersion] = useState<ModVersion | null>(null);
  const [installedResources, setInstalledResources] = useState<{ source: ModSource; project_id?: string }[]>([]);

  const resourceType: ModProjectType = detail?.project_type ?? 'mod';
  const { installingVersionId, installedVersionId, installBatch } = useModInstall(instance.id);

  useEffect(() => {
    if (!source || !projectId) return;
    let cancelled = false;
    setLoading(true);
    void (async () => {
      try {
        const project = await api.getModProject(source, projectId);
        if (cancelled) return;
        setDetail(project);
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
    if (!source || !projectId) return;
    let cancelled = false;
    setVersionsLoading(true);
    void (async () => {
      try {
        const res = await api.listModVersions(source, projectId, {
          loader,
          mc_version: mcVersion,
        });
        if (cancelled) return;
        setVersions(res.items ?? []);
      } catch (e) {
        if (cancelled) return;
        message.error(formatModCatalogError(e, t, 'qxmods.versionsFailed'));
        setVersions([]);
      } finally {
        if (!cancelled) setVersionsLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [loader, mcVersion, message, projectId, source, t]);

  useEffect(() => {
    if (!installedVersionId) return;
    const version = versions.find((v) => v.id === installedVersionId) ?? null;
    setLastInstalledVersion(version);
  }, [installedVersionId, versions]);

  useEffect(() => {
    void (async () => {
      try {
        const res = await api.listInstanceResources(instance.id);
        setInstalledResources(res.items ?? []);
      } catch {
        setInstalledResources([]);
      }
    })();
  }, [instance.id, installedVersionId]);

  const installedProjectIds = useMemo(
    () =>
      new Set(
        installedResources
          .filter((r) => r.project_id)
          .map((r) => `${r.source}:${r.project_id}`),
      ),
    [installedResources],
  );

  const mcVersionOptions = useMemo(() => {
    const fromProject = detail?.game_versions ?? [];
    const fromVersions = versions.flatMap((v) => v.game_versions ?? []);
    const unique = [...new Set([instance.mc_version, ...fromProject, ...fromVersions])];
    return unique.map((value) => ({ value, label: value }));
  }, [detail?.game_versions, instance.mc_version, versions]);

  const loaderOptions = useMemo(() => {
    const fromProject = detail?.loaders ?? [];
    const fromVersions = versions.flatMap((v) => v.loaders ?? []);
    const unique = [...new Set([instance.loader, ...fromProject, ...fromVersions])];
    return unique.map((value) => ({ value, label: value }));
  }, [detail?.loaders, instance.loader, versions]);

  const showSyncButton =
    canSync &&
    detail != null &&
    lastInstalledVersion != null &&
    modSupportsServerSync(detail) &&
    resourceType === 'mod';

  const handleInstallClick = (version: ModVersion) => {
    if (!source || !detail) return;
    if (resourceType === 'datapack' || resourceType === 'resourcepack' || resourceType === 'shader') {
      void installBatch([
        {
          source,
          projectId: detail.id,
          projectName: detail.name,
          version,
          resourceType,
          iconUrl: detail.icon_url,
          downloads: detail.downloads,
          fileSize: version.files[0]?.size,
        },
      ]);
      return;
    }
    setPendingVersion(version);
    setDepsOpen(true);
  };

  const handleInstallConfirm = async (items: InstallItem[]) => {
    const enriched = items.map((item) => ({
      ...item,
      iconUrl: item.projectId === detail?.id ? detail?.icon_url : undefined,
      downloads: item.projectId === detail?.id ? detail?.downloads : undefined,
      fileSize: item.version.files[0]?.size,
    }));
    const ok = await installBatch(enriched);
    if (ok) setDepsOpen(false);
  };

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
        <a href={detail.external_url} target="_blank" rel="noreferrer" className="qxmods-detail-external">
          <LinkOutlined /> {t('qxmods.viewOnSource')}
        </a>
      </div>
      <div className="qxmods-detail-header">
        {detail.icon_url ? (
          <img src={detail.icon_url} alt={detail.name} className="qxmods-result-icon" />
        ) : (
          <span className="qxmods-result-icon qxmods-result-icon--placeholder" />
        )}
        <div>
          <Title level={3} className="qxmods-detail-title">
            <Space wrap>
              {detail.name}
              <ModSourceBadge source={detail.source} />
            </Space>
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
      {versionsLoading ? (
        <Spin />
      ) : versions.length === 0 ? (
        <Empty description={t('qxmods.noVersions')} />
      ) : (
        <>
          <Text strong>{t('qxmods.selectVersion')}</Text>
          <ul className="qxmods-version-list">
            {versions.map((version) => (
              <li key={version.id} className="qxmods-version-row">
                <div>
                  <Text>{version.version_number}</Text>
                  {version.game_versions?.length ? (
                    <Text type="secondary" className="qxmods-version-meta">
                      {' '}
                      · MC {version.game_versions.join(', ')}
                    </Text>
                  ) : null}
                </div>
                <Button
                  type="primary"
                  size="small"
                  loading={installingVersionId === version.id}
                  disabled={installingVersionId != null && installingVersionId !== version.id}
                  onClick={() => handleInstallClick(version)}
                >
                  {t('qxmods.install.action')}
                </Button>
              </li>
            ))}
          </ul>
        </>
      )}
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
      {pendingVersion && source && detail ? (
        <ModInstallDepsModal
          open={depsOpen}
          source={source}
          projectId={detail.id}
          projectName={detail.name}
          version={pendingVersion}
          resourceType={resourceType}
          installedProjectIds={installedProjectIds}
          confirming={installingVersionId === pendingVersion.id}
          onCancel={() => setDepsOpen(false)}
          onConfirm={(items) => void handleInstallConfirm(items)}
        />
      ) : null}
    </section>
  );
}
