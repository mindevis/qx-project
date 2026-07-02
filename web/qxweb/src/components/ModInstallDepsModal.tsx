import { useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react';
import { Checkbox, Modal, Spin, Typography } from 'antd';
import {
  api,
  type ModDependency,
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { modalMotionProps } from '@/lib/modal';
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
  onCancel: () => void;
  onConfirm: (items: InstallItem[]) => void;
  confirming: boolean;
};

export type InstallItem = {
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  resourceType: ModProjectType;
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
  onCancel,
  onConfirm,
  confirming,
}: Props) {
  const { t } = useI18n();
  const { instance } = useInstanceMods();
  const [loading, setLoading] = useState(false);
  const [dependencies, setDependencies] = useState<ModDependency[]>([]);
  const [requiredSelected, setRequiredSelected] = useState<Set<string>>(new Set());
  const [optionalSelected, setOptionalSelected] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    setRequiredSelected(new Set());
    setOptionalSelected(new Set());
    void (async () => {
      try {
        const detail = await api.getModVersion(source, projectId, version.id, {
          loader: instance.loader,
          mc_version: instance.mc_version,
        });
        if (cancelled) return;
        const nextDependencies = detail.dependencies ?? [];
        setDependencies(nextDependencies);
        setRequiredSelected(defaultRequiredSelection(nextDependencies, installedProjectIds));
      } catch {
        if (!cancelled) {
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
  }, [installedProjectIds, instance.loader, instance.mc_version, open, projectId, source, version.id]);

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

  const isDependencySelected = (dep: ModDependency) => {
    if (!dep.project_id) return false;
    if (isDependencyInstalled(dep)) return false;
    if (dep.dependency_type === 'required') {
      return requiredSelected.has(dep.project_id);
    }
    return optionalSelected.has(dep.project_id);
  };

  const buildInstallItems = (): InstallItem[] => {
    const items: InstallItem[] = [];
    const addDep = (dep: ModDependency) => {
      if (!dep.project_id || !dep.version_id || !dep.download_url || !dep.filename) return;
      if (isDependencyInstalled(dep)) return;
      items.push({
        source: dep.source,
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
      if (requiredSelected.has(dep.project_id)) addDep(dep);
    }
    for (const dep of optional) {
      if (optionalSelected.has(dep.project_id)) addDep(dep);
    }
    items.push({
      source,
      projectId,
      projectName,
      version,
      resourceType,
    });
    return items;
  };

  const selectedUnresolved = [...required, ...optional].filter(
    (dep) => isDependencySelected(dep) && (!dep.download_url || !dep.filename || !dep.version_id),
  );
  const canInstall = selectedUnresolved.length === 0;

  const renderDependencyCheckbox = (
    dep: ModDependency,
    selected: Set<string>,
    setSelected: Dispatch<SetStateAction<Set<string>>>,
  ) => {
    const installed = isDependencyInstalled(dep);
    const unresolved = !dep.download_url || !dep.filename || !dep.version_id;
    return (
      <li key={dependencyProjectKey(dep) ?? dep.project_name} className="qxmods-deps-item">
        <Checkbox
          checked={installed || selected.has(dep.project_id)}
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

  return (
    <Modal
      {...modalMotionProps}
      title={t('qxmods.deps.title')}
      open={open}
      onCancel={onCancel}
      onOk={() => onConfirm(buildInstallItems())}
      okText={t('qxmods.deps.installAll')}
      cancelText={t('common.cancel')}
      confirmLoading={confirming}
      okButtonProps={{ disabled: loading || !canInstall }}
    >
      {loading ? (
        <Spin />
      ) : (
        <>
          <Paragraph type="secondary">{t('qxmods.deps.intro', { name: projectName })}</Paragraph>
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
        </>
      )}
    </Modal>
  );
}
