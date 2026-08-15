import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Modal, Select, Tag } from 'antd';
import { CheckCircleOutlined } from '@ant-design/icons';
import {
  api,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { ModInstallDepsWizard } from '@/components/ModInstallDepsWizard';
import type { InstallItem } from '@/components/ModInstallDepsModal';
import { ServerOnlyInstallModal } from '@/components/ServerOnlyInstallModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useI18n } from '@/i18n/I18nContext';
import { useModInstall } from '@/hooks/useModInstall';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { isServerOnlyMod } from '@/lib/modSync';
import { fetchModProjectIcons } from '@/lib/instanceResourceIcons';
import {
  cachedGetModVersion,
  cachedListModVersions,
  clearModVersionListCache,
} from '@/lib/modCatalogCache';
import { isDependencyResolved, loadModDirectDependencies } from '@/lib/modCatalogDeps';
import { selectLatestCompatibleVersion } from '@/lib/selectLatestModVersion';
import { useMessage } from '@/hooks/useMessage';

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
  eagerVersions?: boolean;
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
  eagerVersions = true,
  onInstalled,
  onUninstalled,
}: ModCatalogInstallControlsProps) {
  const { t } = useI18n();
  const message = useMessage();
  const { instance } = useInstanceMods();
  const { installingVersionId, installBatch } = useModInstall(instance.id);

  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [selectedVersionId, setSelectedVersionId] = useState<string>();
  const [loadingVersions, setLoadingVersions] = useState(eagerVersions);
  const [versionsLoaded, setVersionsLoaded] = useState(false);
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
    setLoadingVersions(true);
    try {
      const items = await cachedListModVersions(source, projectId, {
        loader,
        mc_version: mcVersion,
      });
      const latest = selectLatestCompatibleVersion(items, loader, mcVersion);
      setVersions(items);
      setSelectedVersionId((prev) => prev ?? latest?.id);
      setVersionsLoaded(true);
      return items;
    } catch (e) {
      message.error(formatModCatalogError(e, t, 'qxmods.versionsFailed'));
      setVersions([]);
      setSelectedVersionId(undefined);
      setVersionsLoaded(true);
      return [] as ModVersion[];
    } finally {
      setLoadingVersions(false);
    }
  }, [loader, mcVersion, message, projectId, source, t]);

  useEffect(() => {
    setVersions([]);
    setSelectedVersionId(undefined);
    setVersionsLoaded(false);
    setLoadingVersions(eagerVersions);
    if (eagerVersions) {
      void loadVersions();
    }
  }, [eagerVersions, loadVersions, source, projectId, loader, mcVersion]);

  const versionOptions = useMemo(
    () =>
      versions.map((version) => ({
        value: version.id,
        label: version.version_number,
      })),
    [versions],
  );

  const resolveVersionForInstall = useCallback(
    async (
      version: ModVersion,
      itemSource: ModSource = source,
      itemProjectId: string = projectId,
    ): Promise<ModVersion | null> => {
      if (version.files[0]?.url) {
        return version;
      }
      try {
        const detail = await cachedGetModVersion(itemSource, itemProjectId, version.id, {
          loader,
          mc_version: mcVersion,
        });
        if (!detail.files[0]?.url) {
          message.error(t('qxmods.install.noFile'));
          return null;
        }
        return detail;
      } catch (e) {
        message.error(formatModCatalogError(e, t, 'qxmods.install.failed'));
        return null;
      }
    },
    [loader, mcVersion, message, projectId, source, t],
  );

  const installCatalogItem = async (version: ModVersion, extraItems: InstallItem[] = []) => {
    const resolved = await resolveVersionForInstall(version);
    if (!resolved) return false;
    const ok = await installBatch([
      ...extraItems,
      {
        source,
        projectId,
        projectName,
        version: resolved,
        resourceType: projectType,
        iconUrl,
        downloads,
        fileSize: resolved.files[0]?.size,
      },
    ]);
    if (ok) onInstalled?.(resolved);
    return ok;
  };

  const runInstall = (version: ModVersion) => {
    if (projectType === 'datapack' || projectType === 'resourcepack' || projectType === 'shader') {
      void installCatalogItem(version);
      return;
    }
    void (async () => {
      const resolved = await resolveVersionForInstall(version);
      if (!resolved) return;

      if (
        projectType === 'mod' &&
        isServerOnlyMod({ client_side: clientSide, server_side: serverSide })
      ) {
        setPendingVersion(resolved);
        setServerOnlyOpen(true);
        return;
      }

      if (projectType !== 'mod') {
        await installCatalogItem(resolved);
        return;
      }

      let directRequired: Awaited<ReturnType<typeof loadModDirectDependencies>>['required'] = [];
      let optional: Awaited<ReturnType<typeof loadModDirectDependencies>>['optional'] = [];
      try {
        const loaded = await loadModDirectDependencies(source, projectId, resolved, {
          loader,
          mcVersion,
        });
        directRequired = loaded.required;
        optional = loaded.optional;
      } catch (e) {
        message.error(formatModCatalogError(e, t, 'qxmods.versionsFailed'));
        return;
      }

      const pendingRequired = directRequired.filter(
        (dep) => dep.project_id && !installedProjectIds.has(`${dep.source}:${dep.project_id}`),
      );
      const pendingOptional = optional.filter(
        (dep) => dep.project_id && !installedProjectIds.has(`${dep.source}:${dep.project_id}`),
      );
      const unresolvedRequired = pendingRequired.filter((dep) => !isDependencyResolved(dep));

      if (unresolvedRequired.length > 0) {
        setPendingVersion(resolved);
        setDepsOpen(true);
        return;
      }

      if (pendingRequired.length === 0 && pendingOptional.length === 0) {
        await installCatalogItem(resolved);
        return;
      }

      setPendingVersion(resolved);
      setDepsOpen(true);
    })();
  };

  const handleInstallConfirm = async (items: InstallItem[]) => {
    const resolvedItems: InstallItem[] = [];
    for (const item of items) {
      const resolvedVersion = await resolveVersionForInstall(
        item.version,
        item.source,
        item.projectId,
      );
      if (!resolvedVersion) return;
      resolvedItems.push({ ...item, version: resolvedVersion });
    }
    const iconMap = await fetchModProjectIcons(
      resolvedItems.map((item) => ({ source: item.source, projectId: item.projectId })),
    );
    if (iconUrl) {
      iconMap.set(projectKey, iconUrl);
    }
    const enriched = resolvedItems.map((item) => ({
      ...item,
      iconUrl: iconMap.get(`${item.source}:${item.projectId}`),
      downloads: item.projectId === projectId ? downloads : undefined,
      fileSize: item.version.files[0]?.size,
    }));
    const ok = await installBatch(enriched);
    if (ok) {
      setDepsOpen(false);
      const primary = resolvedItems[resolvedItems.length - 1];
      if (primary) onInstalled?.(primary.version);
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

  const ensureVersionAndInstall = () => {
    void (async () => {
      let version = selectedVersion;
      if (!version) {
        const items = versionsLoaded ? versions : await loadVersions();
        version = selectLatestCompatibleVersion(items, loader, mcVersion);
      }
      if (version) {
        runInstall(version);
      }
    })();
  };

  const installing = selectedVersion != null && installingVersionId === selectedVersion.id;
  const disabled = (installingVersionId != null && !installing) || uninstalling;

  if (versionsLoaded && versions.length === 0) {
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
          onOpenChange={(open) => {
            if (open && !versionsLoaded && !loadingVersions) {
              void loadVersions();
            }
          }}
        />
        <Button
          type="primary"
          size="small"
          className="qxmods-install-action"
          loading={installing || (!versionsLoaded && loadingVersions)}
          disabled={disabled || (versionsLoaded && !selectedVersion)}
          onClick={ensureVersionAndInstall}
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
          <ModInstallDepsWizard
            open={depsOpen}
            source={source}
            projectId={projectId}
            projectName={projectName}
            version={pendingVersion}
            resourceType={projectType}
            installedProjectIds={installedProjectIds}
            confirming={installingVersionId === pendingVersion.id}
            onCancel={() => setDepsOpen(false)}
            onComplete={(items) => void handleInstallConfirm(items)}
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
            preferInstance
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
  clearModVersionListCache();
}
