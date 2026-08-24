import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  type GameServerContentKind,
  type InstanceResource,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { useModCatalog } from '@/components/ModCatalogContext';
import { ModVersionSelect } from '@/components/ModVersionSelect';
import { useI18n } from '@/i18n/I18nContext';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { cachedGetModVersion, cachedListModVersions } from '@/lib/modCatalogCache';
import { contentKindHasSide, gameServerInstallSide } from '@/lib/modSync';
import { filterVersionsForCatalogLoader } from '@/lib/selectLatestModVersion';
import { useMessage } from '@/hooks/useMessage';

function resourceAsVersion(resource: InstanceResource): ModVersion {
  return {
    id: resource.version_id || `installed:${resource.filename}`,
    version_number: resource.version_number || resource.filename,
    version_type: resource.version_type,
    files: [],
  };
}

type CatalogInstalledResource = InstanceResource & {
  project_id: string;
  source: Exclude<ModSource, 'upload'>;
};

export function canChangeInstalledContentVersion(
  resource?: InstanceResource,
): resource is CatalogInstalledResource {
  return Boolean(
    resource &&
      resource.project_id &&
      resource.source &&
      resource.source !== 'upload',
  );
}

export function InstalledContentVersionSelect({
  kind,
  resource,
  diskFilename,
  disabled,
  onUpdated,
}: {
  kind: GameServerContentKind;
  resource: InstanceResource;
  diskFilename: string;
  disabled?: boolean;
  onUpdated: () => void;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const { loader, mcVersion, installingVersionId, installBatch } = useModCatalog();
  const current = useMemo(
    () => resourceAsVersion(resource),
    [resource.filename, resource.version_id, resource.version_number, resource.version_type],
  );
  const [versions, setVersions] = useState<ModVersion[]>([current]);
  const [loadingVersions, setLoadingVersions] = useState(false);
  const [versionsLoaded, setVersionsLoaded] = useState(false);
  const [pendingVersionId, setPendingVersionId] = useState<string>();
  const replacingRef = useRef(false);

  useEffect(() => {
    if (pendingVersionId) {
      if (pendingVersionId === current.id) {
        setPendingVersionId(undefined);
      }
      return;
    }
    setVersions((prev) => (prev.some((item) => item.id === current.id) ? prev : [current, ...prev]));
  }, [current, pendingVersionId]);

  const selectedId =
    pendingVersionId ?? (versions.some((item) => item.id === current.id) ? current.id : versions[0]?.id);

  const loadVersions = useCallback(async () => {
    if (!resource.project_id) return;
    setLoadingVersions(true);
    try {
      const items = filterVersionsForCatalogLoader(
        await cachedListModVersions(resource.source as ModSource, resource.project_id, {
          loader: loader || undefined,
          mc_version: mcVersion,
        }),
        loader || undefined,
      );
      setVersions(items.some((item) => item.id === current.id) ? items : [current, ...items]);
      setVersionsLoaded(true);
    } catch (e) {
      message.error(formatModCatalogError(e, t, 'qxmods.versionsFailed'));
      setVersions([current]);
      setVersionsLoaded(true);
    } finally {
      setLoadingVersions(false);
    }
  }, [current, loader, mcVersion, message, resource.project_id, resource.source, t]);

  const handleChange = async (versionId: string) => {
    if (
      replacingRef.current ||
      !resource.project_id ||
      versionId === current.id ||
      versionId === pendingVersionId
    ) {
      return;
    }
    const chosen = versions.find((item) => item.id === versionId);
    if (!chosen) return;
    replacingRef.current = true;
    setPendingVersionId(versionId);
    try {
      let version = chosen;
      if (!version.files[0]?.url) {
        version = await cachedGetModVersion(
          resource.source as ModSource,
          resource.project_id,
          version.id,
          { loader: loader || undefined, mc_version: mcVersion },
        );
      }
      if (!version.files[0]?.url) {
        message.error(t('qxmods.install.noFile'));
        setPendingVersionId(undefined);
        return;
      }
      const ok = await installBatch([
        {
          source: resource.source as ModSource,
          projectId: resource.project_id,
          projectName: resource.project_name,
          version,
          resourceType: resource.resource_type,
          iconUrl: resource.icon_url,
          downloads: resource.downloads,
          fileSize: version.files[0]?.size ?? resource.file_size,
          side: contentKindHasSide(kind) ? gameServerInstallSide(resource.side_override) : undefined,
          replaceFilename: diskFilename,
        },
      ]);
      if (ok) {
        onUpdated();
      } else {
        setPendingVersionId(undefined);
      }
    } catch (e) {
      setPendingVersionId(undefined);
      message.error(formatModCatalogError(e, t, 'qxmods.install.failed'));
    } finally {
      replacingRef.current = false;
    }
  };

  return (
    <ModVersionSelect
      versions={versions}
      value={selectedId}
      loading={loadingVersions || installingVersionId != null}
      disabled={disabled || installingVersionId != null}
      size="small"
      className="qxmods-installed-version-select"
      placeholder={t('qxmods.selectVersion')}
      ariaLabel={t('gameServerDetail.content.changeVersionAria', {
        name: resource.project_name || resource.filename,
      })}
      onChange={(versionId) => void handleChange(versionId)}
      onOpenChange={(open) => {
        if (open && !loadingVersions && !versionsLoaded) {
          void loadVersions();
        }
      }}
    />
  );
}
