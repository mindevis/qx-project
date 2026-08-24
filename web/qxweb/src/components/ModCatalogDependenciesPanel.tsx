import { useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from 'antd';
import { CheckCircleOutlined } from '@ant-design/icons';
import {
  type ModDependency,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { useModCatalog } from '@/components/ModCatalogContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { formatModCatalogError } from '@/lib/modCatalogError';
import {
  dependencyProjectKey,
  isDependencyResolved,
  loadModDirectDependencies,
} from '@/lib/modCatalogDeps';

const { Text } = Typography;

export function ModCatalogDependenciesPanel({
  source,
  projectId,
  version,
  resourceType,
  installedProjectIds,
  onInstalled,
}: {
  source: ModSource;
  projectId: string;
  version?: ModVersion;
  resourceType: ModProjectType;
  installedProjectIds: Set<string>;
  onInstalled?: () => void;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const { loader, mcVersion, installingVersionId, installBatch } = useModCatalog();
  const [loading, setLoading] = useState(false);
  const [required, setRequired] = useState<ModDependency[]>([]);
  const [optional, setOptional] = useState<ModDependency[]>([]);

  useEffect(() => {
    if (!version) {
      setRequired([]);
      setOptional([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    void (async () => {
      try {
        const loaded = await loadModDirectDependencies(source, projectId, version, {
          loader,
          mcVersion,
        });
        if (cancelled) return;
        setRequired(loaded.required);
        setOptional(loaded.optional);
      } catch (e) {
        if (!cancelled) {
          message.error(formatModCatalogError(e, t, 'qxmods.versionsFailed'));
          setRequired([]);
          setOptional([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [loader, mcVersion, message, projectId, source, t, version]);

  const installDependency = async (dep: ModDependency) => {
    if (!dep.project_id || !isDependencyResolved(dep)) return;
    const ok = await installBatch([
      {
        source: dep.source as ModSource,
        projectId: dep.project_id,
        projectName: dep.project_name ?? dep.project_id,
        version: {
          id: dep.version_id!,
          version_number: dep.version_number ?? dep.version_id!,
          files: [{ filename: dep.filename!, url: dep.download_url!, size: dep.file_size }],
        },
        resourceType,
      },
    ]);
    if (ok) onInstalled?.();
  };

  if (!version) return null;
  if (loading) {
    return (
      <div className="qxmods-detail-deps">
        <Spin />
      </div>
    );
  }

  const renderRow = (dep: ModDependency) => {
    const key = dependencyProjectKey(dep) ?? dep.project_name ?? dep.project_id;
    const installed = Boolean(key && installedProjectIds.has(`${dep.source}:${dep.project_id}`));
    const unresolved = !isDependencyResolved(dep);
    const name = dep.project_name ?? dep.project_id;
    const installing = installingVersionId != null && installingVersionId === dep.version_id;
    return (
      <li key={key} className="qxmods-detail-deps-item">
        <div className="qxmods-detail-deps-copy">
          <span className="qxmods-detail-deps-name">{name}</span>
          {dep.version_number ? (
            <Text type="secondary" className="qxmods-detail-deps-version">
              {dep.version_number}
            </Text>
          ) : null}
          {unresolved ? (
            <Text type="danger">({t('qxmods.deps.unresolved')})</Text>
          ) : null}
        </div>
        {installed ? (
          <Tag icon={<CheckCircleOutlined />} color="success" className="qxmods-installed-badge">
            {t('qxmods.deps.installed')}
          </Tag>
        ) : (
          <Button
            type="primary"
            size="small"
            disabled={unresolved || installingVersionId != null}
            loading={installing}
            aria-label={t('qxmods.deps.installAria', { name })}
            onClick={() => void installDependency(dep)}
          >
            {t('qxmods.install.action')}
          </Button>
        )}
      </li>
    );
  };

  return (
    <div className="qxmods-detail-deps">
      {required.length > 0 ? (
        <>
          <Text strong>{t('qxmods.deps.required')}</Text>
          <ul className="qxmods-deps-list">{required.map(renderRow)}</ul>
        </>
      ) : (
        <Text type="secondary">{t('qxmods.deps.noRequired')}</Text>
      )}
      {optional.length > 0 ? (
        <>
          <Text strong className="qxmods-deps-optional-title">
            {t('qxmods.deps.optional')}
          </Text>
          <ul className="qxmods-deps-list">{optional.map(renderRow)}</ul>
        </>
      ) : null}
    </div>
  );
}
