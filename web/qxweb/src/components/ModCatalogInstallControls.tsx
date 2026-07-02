import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Modal, Select, Spin, Tag } from 'antd';
import { CheckCircleOutlined } from '@ant-design/icons';
import {
  api,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { ModInstallDepsModal, type InstallItem } from '@/components/ModInstallDepsModal';
import { ServerOnlyInstallModal } from '@/components/ServerOnlyInstallModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useModInstall } from '@/hooks/useModInstall';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { isServerOnlyMod } from '@/lib/modSync';
import { fetchModProjectIcons } from '@/lib/instanceResourceIcons';
import { useMessage } from '@/hooks/useMessage';

const versionCache = new Map<string, ModVersion[]>();

function cacheKey(
  source: ModSource,
  projectId: string,
  loader: string | undefined,
  mcVersion: string,
) {
  return `${source}:${projectId}:${loader ?? ''}:${mcVersion}`;
}

export type ModCatalogInstallControlsProps = {
  source: ModSource;
  projectId: string;
  projectName: string;
  projectType: ModProjectType;
  iconUrl?: string;
  downloads?: number;
  clientSide?: string;
  serverSide?: string;
  loader?: string;
  mcVersion: string;
  installedProjectIds: Set<string>;
  layout?: 'inline' | 'stacked';
  selectClassName?: string;
  onInstalled?: (version: ModVersion) => void;
  onUninstalled?: () => void;
};

export function ModCatalogInstallControls({
  source,
  projectId,
  projectName,
  projectType,
  iconUrl,
  downloads,
  clientSide,
  serverSide,
  loader,
  mcVersion,
  installedProjectIds,
  layout = 'inline',
  selectClassName,
  onInstalled,
  onUninstalled,
}: ModCatalogInstallControlsProps) {
  const { t } = useI18n();
  const message = useMessage();
  const { instance } = useInstanceMods();
  const { installingVersionId, installBatch } = useModInstall(instance.id);

  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [selectedVersionId, setSelectedVersionId] = useState<string>();
  const [loadingVersions, setLoadingVersions] = useState(true);
  const [depsOpen, setDepsOpen] = useState(false);
  const [serverOnlyOpen, setServerOnlyOpen] = useState(false);
  const [pendingVersion, setPendingVersion] = useState<ModVersion | null>(null);
  const [uninstalling, setUninstalling] = useState(false);

  const projectKey = `${source}:${projectId}`;
  const isInstalled = installedProjectIds.has(projectKey);

  const selectedVersion = useMemo(
    () => versions.find((v) => v.id === selectedVersionId) ?? versions[0],
    [selectedVersionId, versions],
  );

  const loadVersions = useCallback(async () => {
    const key = cacheKey(source, projectId, loader, mcVersion);
    const cached = versionCache.get(key);
    if (cached) {
      setVersions(cached);
      setSelectedVersionId((prev) => prev ?? cached[0]?.id);
      setLoadingVersions(false);
      return;
    }
    setLoadingVersions(true);
    try {
      const res = await api.listModVersions(source, projectId, {
        loader,
        mc_version: mcVersion,
      });
      const items = res.items ?? [];
      versionCache.set(key, items);
      setVersions(items);
      setSelectedVersionId((prev) => prev ?? items[0]?.id);
    } catch (e) {
      message.error(formatModCatalogError(e, t, 'qxmods.versionsFailed'));
      setVersions([]);
      setSelectedVersionId(undefined);
    } finally {
      setLoadingVersions(false);
    }
  }, [loader, mcVersion, message, projectId, source, t]);

  useEffect(() => {
    setVersions([]);
    setSelectedVersionId(undefined);
    setLoadingVersions(true);
    void loadVersions();
  }, [loadVersions, source, projectId, loader, mcVersion]);

  const versionOptions = useMemo(
    () =>
      versions.map((version) => ({
        value: version.id,
        label: version.version_number,
      })),
    [versions],
  );

  const runInstall = (version: ModVersion) => {
    if (projectType === 'datapack' || projectType === 'resourcepack' || projectType === 'shader') {
      void (async () => {
        const ok = await installBatch([
          {
            source,
            projectId,
            projectName,
            version,
            resourceType: projectType,
            iconUrl,
            downloads,
            fileSize: version.files[0]?.size,
          },
        ]);
        if (ok) onInstalled?.(version);
      })();
      return;
    }
    if (
      projectType === 'mod' &&
      isServerOnlyMod({ client_side: clientSide, server_side: serverSide })
    ) {
      setPendingVersion(version);
      setServerOnlyOpen(true);
      return;
    }
    setPendingVersion(version);
    setDepsOpen(true);
  };

  const handleInstallConfirm = async (items: InstallItem[]) => {
    const iconMap = await fetchModProjectIcons(
      items.map((item) => ({ source: item.source, projectId: item.projectId })),
    );
    if (iconUrl) {
      iconMap.set(projectKey, iconUrl);
    }
    const enriched = items.map((item) => ({
      ...item,
      iconUrl: iconMap.get(`${item.source}:${item.projectId}`),
      downloads: item.projectId === projectId ? downloads : undefined,
      fileSize: item.version.files[0]?.size,
    }));
    const ok = await installBatch(enriched);
    if (ok) {
      setDepsOpen(false);
      if (pendingVersion) onInstalled?.(pendingVersion);
    }
  };

  const handleUninstall = () => {
    Modal.confirm({
      title: t('qxmods.uninstall.confirmTitle'),
      content: t('qxmods.uninstall.confirmBody', { name: projectName }),
      okText: t('qxmods.uninstall.action'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        setUninstalling(true);
        try {
          const resources = await api.listInstanceResources(instance.id);
          const match = (resources.items ?? []).find(
            (item) =>
              item.source === source &&
              item.project_id === projectId &&
              item.resource_type === projectType,
          );
          await api.deleteInstanceResource(instance.id, {
            source,
            project_id: projectId,
            filename: match?.filename,
            resource_type: projectType,
          });
          message.success(t('qxmods.uninstall.completed'));
          onUninstalled?.();
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('qxmods.uninstall.failed'));
          throw e;
        } finally {
          setUninstalling(false);
        }
      },
    });
  };

  const installing = selectedVersion != null && installingVersionId === selectedVersion.id;
  const disabled = (installingVersionId != null && !installing) || uninstalling;

  if (loadingVersions && versions.length === 0) {
    return <Spin size="small" />;
  }

  if (versions.length === 0) {
    return (
      <div className="qxmods-install-controls qxmods-install-controls--inline">
        {isInstalled ? (
          <Tag icon={<CheckCircleOutlined />} color="success" className="qxmods-installed-badge">
            {t('qxmods.installed.badge')}
          </Tag>
        ) : null}
        <span className="qxmods-install-no-versions">{t('qxmods.noVersions')}</span>
        {isInstalled ? (
          <Button size="small" danger loading={uninstalling} disabled={disabled} onClick={handleUninstall}>
            {t('qxmods.uninstall.action')}
          </Button>
        ) : null}
      </div>
    );
  }

  return (
    <>
      <div className={`qxmods-install-controls qxmods-install-controls--${layout}`}>
        {isInstalled ? (
          <Tag icon={<CheckCircleOutlined />} color="success" className="qxmods-installed-badge">
            {t('qxmods.installed.badge')}
          </Tag>
        ) : null}
        <Select
          showSearch
          optionFilterProp="label"
          placeholder={t('qxmods.selectVersion')}
          className={selectClassName ?? 'qxmods-install-version-select'}
          loading={loadingVersions}
          disabled={disabled}
          value={selectedVersion?.id}
          options={versionOptions}
          onChange={setSelectedVersionId}
          onDropdownVisibleChange={(open) => {
            if (open && versions.length === 0) {
              void loadVersions();
            }
          }}
        />
        <Button
          type="primary"
          size="small"
          className="qxmods-install-action"
          loading={installing}
          disabled={disabled || !selectedVersion}
          onClick={() => selectedVersion && runInstall(selectedVersion)}
        >
          {t('qxmods.install.action')}
        </Button>
        {isInstalled ? (
          <Button size="small" danger loading={uninstalling} disabled={disabled} onClick={handleUninstall}>
            {t('qxmods.uninstall.action')}
          </Button>
        ) : null}
      </div>
      {pendingVersion ? (
        <>
          <ModInstallDepsModal
            open={depsOpen}
            source={source}
            projectId={projectId}
            projectName={projectName}
            version={pendingVersion}
            resourceType={projectType}
            installedProjectIds={installedProjectIds}
            confirming={installingVersionId === pendingVersion.id}
            onCancel={() => setDepsOpen(false)}
            onConfirm={(items) => void handleInstallConfirm(items)}
          />
          <ServerOnlyInstallModal
            open={serverOnlyOpen}
            source={source}
            projectId={projectId}
            projectName={projectName}
            version={pendingVersion}
            projectType={projectType}
            instanceLoader={instance.loader}
            instanceMcVersion={instance.mc_version}
            onClose={() => setServerOnlyOpen(false)}
            onInstallToInstance={() => {
              setServerOnlyOpen(false);
              setDepsOpen(true);
            }}
          />
        </>
      ) : null}
    </>
  );
}

export function clearModVersionCache() {
  versionCache.clear();
}
