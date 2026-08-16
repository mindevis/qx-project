import { useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react';
import { Alert, Checkbox, Modal, Spin, Typography } from 'antd';
import {
  type ModDependency,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useModCatalog } from '@/components/ModCatalogContext';
import { useMessage } from '@/hooks/useMessage';
import { formatModCatalogError } from '@/lib/modCatalogError';
import { modalMotionProps } from '@/lib/modal';
import {
  loadModDirectDependencies,
  unresolvedSelectedDependencies,
} from '@/lib/modCatalogDeps';
import './InstanceResourcesPanel.css';

const { Text, Paragraph } = Typography;

type Props = {
  open: boolean;
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  resourceType: ModProjectType;
  installedProjectIds: Set<string>;
  nested?: boolean;
  hasMoreSteps?: boolean;
  onCancel: () => void;
  onConfirm: (result: DepsModalConfirmResult) => void;
  confirming: boolean;
};

export type InstallItem = {
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  resourceType: ModProjectType;
};

export type DepsModalConfirmResult = {
  items: InstallItem[];
  selectedRequired: ModDependency[];
};

function dependencyProjectKey(dep: ModDependency): string | undefined {
  if (!dep.project_id) return undefined;
  return `${dep.source}:${dep.project_id}`;
}

export function defaultRequiredSelection(
  dependencies: ModDependency[],
  installedProjectIds: Set<string>,
): Set<string> {
  return new Set(
    dependencies
      .filter((dep) => dep.dependency_type === 'required' && dep.project_id)
      .filter((dep) => !installedProjectIds.has(`${dep.source}:${dep.project_id}`))
      .map((dep) => dep.project_id),
  );
}

export function ModInstallDepsModal({
  open,
  source,
  projectId,
  projectName,
  version,
  resourceType,
  installedProjectIds,
  nested = false,
  hasMoreSteps = false,
  onCancel,
  onConfirm,
  confirming,
}: Props) {
  const { t } = useI18n();
  const message = useMessage();
  const { loader, mcVersion } = useModCatalog();
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [installVersion, setInstallVersion] = useState<ModVersion>(version);
  const [dependencies, setDependencies] = useState<ModDependency[]>([]);
  const [requiredSelected, setRequiredSelected] = useState<Set<string>>(new Set());
  const [optionalSelected, setOptionalSelected] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    setLoadError(null);
    setInstallVersion(version);
    setRequiredSelected(new Set());
    setOptionalSelected(new Set());
    void (async () => {
      try {
        const { installVersion: detail, required, optional } = await loadModDirectDependencies(
          source,
          projectId,
          version,
          { loader, mcVersion },
        );
        if (cancelled) return;
        if (!detail.files[0]?.url) {
          setLoadError(t('qxmods.install.noFile'));
          setDependencies([]);
          setRequiredSelected(new Set());
          return;
        }
        setInstallVersion(detail);
        const nextDependencies = [...required, ...optional];
        setDependencies(nextDependencies);
        setRequiredSelected(defaultRequiredSelection(required, installedProjectIds));
      } catch (e) {
        if (!cancelled) {
          const msg = formatModCatalogError(e, t, 'qxmods.versionsFailed');
          setLoadError(msg);
          message.error(msg);
          setDependencies([]);
          setRequiredSelected(new Set());
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [installedProjectIds, loader, mcVersion, message, open, projectId, source, t, version]);

  const required = useMemo(
    () => dependencies.filter((d) => d.dependency_type === 'required'),
    [dependencies],
  );
  const optional = useMemo(
    () => dependencies.filter((d) => d.dependency_type === 'optional'),
    [dependencies],
  );

  const isDependencyInstalled = (dep: ModDependency) => {
    const key = dependencyProjectKey(dep);
    return Boolean(key && installedProjectIds.has(key));
  };

  const buildInstallItems = (): InstallItem[] => {
    const items: InstallItem[] = [];
    const addDep = (dep: ModDependency) => {
      if (!dep.project_id || !dep.version_id || !dep.filename || !dep.download_url) return;
      if (isDependencyInstalled(dep)) return;
      items.push({
        source: dep.source as ModSource,
        projectId: dep.project_id,
        projectName: dep.project_name ?? dep.project_id,
        version: {
          id: dep.version_id,
          version_number: dep.version_number ?? dep.version_id,
          files: [{ filename: dep.filename, url: dep.download_url, size: dep.file_size }],
        },
        resourceType: 'mod',
      });
    };
    for (const dep of required) {
      if (dep.project_id && requiredSelected.has(dep.project_id)) addDep(dep);
    }
    for (const dep of optional) {
      if (dep.project_id && optionalSelected.has(dep.project_id)) addDep(dep);
    }
    items.push({
      source,
      projectId,
      projectName,
      version: installVersion,
      resourceType,
    });
    return items;
  };

  const selectedRequired = useMemo(() => {
    return required.filter((dep) => {
      if (!dep.project_id || !requiredSelected.has(dep.project_id)) return false;
      const key = dependencyProjectKey(dep);
      return !(key && installedProjectIds.has(key));
    });
  }, [installedProjectIds, required, requiredSelected]);

  const selectedUnresolved = unresolvedSelectedDependencies(
    dependencies,
    installedProjectIds,
    requiredSelected,
    optionalSelected,
  );
  const canInstall = !loadError && selectedUnresolved.length === 0 && Boolean(installVersion.files[0]?.url);

  const renderDependencyCheckbox = (
    dep: ModDependency,
    selected: Set<string>,
    setSelected: Dispatch<SetStateAction<Set<string>>>,
  ) => {
    const installed = isDependencyInstalled(dep);
    const unresolved = !dep.project_id || !dep.version_id || !dep.filename || !dep.download_url;
    return (
      <li key={dependencyProjectKey(dep) ?? dep.project_name} className="qxmods-deps-item">
        <Checkbox
          checked={installed || (dep.project_id ? selected.has(dep.project_id) : false)}
          disabled={installed || unresolved}
          onChange={(e) => {
            if (!dep.project_id) return;
            setSelected((prev) => {
              const next = new Set(prev);
              if (e.target.checked) next.add(dep.project_id);
              else next.delete(dep.project_id);
              return next;
            });
          }}
        >
          {dep.project_name ?? dep.project_id}{' '}
          {installed ? (
            <Text type="success">({t('qxmods.deps.installed')})</Text>
          ) : unresolved ? (
            <Text type="danger">({t('qxmods.deps.unresolved')})</Text>
          ) : null}
        </Checkbox>
      </li>
    );
  };

  const title = nested ? t('qxmods.deps.titleFor', { name: projectName }) : t('qxmods.deps.title');
  const intro = nested
    ? t('qxmods.deps.nestedIntro', { name: projectName })
    : t('qxmods.deps.intro', { name: projectName });
  const okText = hasMoreSteps ? t('qxmods.deps.continue') : t('qxmods.deps.installAll');

  return (
    <Modal
      {...modalMotionProps}
      title={title}
      open={open}
      onCancel={onCancel}
      onOk={() => onConfirm({ items: buildInstallItems(), selectedRequired })}
      okText={okText}
      cancelText={t('common.cancel')}
      confirmLoading={confirming}
      okButtonProps={{ disabled: loading || !canInstall }}
    >
      {loading ? (
        <Spin />
      ) : (
        <>
          <Paragraph type="secondary">{intro}</Paragraph>
          {loadError ? (
            <Alert type="error" showIcon className="qxmods-deps-blocked" title={loadError} />
          ) : null}
          {required.length > 0 ? (
            <>
              <Text strong>{t('qxmods.deps.required')}</Text>
              <ul className="qxmods-deps-list">
                {required.map((dep) => renderDependencyCheckbox(dep, requiredSelected, setRequiredSelected))}
              </ul>
            </>
          ) : (
            <Text type="secondary">{t('qxmods.deps.noRequired')}</Text>
          )}
          {optional.length > 0 ? (
            <>
              <Text strong className="qxmods-deps-optional-title">
                {t('qxmods.deps.optional')}
              </Text>
              <ul className="qxmods-deps-list">
                {optional.map((dep) => renderDependencyCheckbox(dep, optionalSelected, setOptionalSelected))}
              </ul>
            </>
          ) : null}
          {!canInstall ? (
            <Alert
              type="warning"
              showIcon
              className="qxmods-deps-blocked"
              title={t('qxmods.deps.unresolvedBlocked')}
            />
          ) : null}
        </>
      )}
    </Modal>
  );
}
