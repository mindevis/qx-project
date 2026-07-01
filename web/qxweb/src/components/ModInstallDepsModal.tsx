import { useEffect, useMemo, useState } from 'react';
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
  const [optionalSelected, setOptionalSelected] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    setOptionalSelected(new Set());
    void (async () => {
      try {
        const detail = await api.getModVersion(source, projectId, version.id, {
          loader: instance.loader,
          mc_version: instance.mc_version,
        });
        if (cancelled) return;
        setDependencies(detail.dependencies ?? []);
      } catch {
        if (!cancelled) setDependencies([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [instance.loader, instance.mc_version, open, projectId, source, version.id]);

  const required = useMemo(
    () => dependencies.filter((d) => d.dependency_type === 'required'),
    [dependencies],
  );
  const optional = useMemo(
    () => dependencies.filter((d) => d.dependency_type === 'optional'),
    [dependencies],
  );

  const missingRequired = required.filter(
    (d) => d.project_id && !installedProjectIds.has(`${d.source}:${d.project_id}`),
  );

  const buildInstallItems = (): InstallItem[] => {
    const items: InstallItem[] = [];
    const addDep = (dep: ModDependency) => {
      if (!dep.project_id || !dep.version_id || !dep.download_url || !dep.filename) return;
      if (installedProjectIds.has(`${dep.source}:${dep.project_id}`)) return;
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
    for (const dep of required) addDep(dep);
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

  const canInstall =
    missingRequired.every((d) => d.version_id && d.download_url && d.filename) || missingRequired.length === 0;

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
                {required.map((dep) => {
                  const installed = dep.project_id
                    ? installedProjectIds.has(`${dep.source}:${dep.project_id}`)
                    : false;
                  return (
                    <li key={`${dep.source}:${dep.project_id ?? dep.project_name}`} className="qxmods-deps-item">
                      <Text>
                        {dep.project_name ?? dep.project_id}{' '}
                        {installed ? (
                          <Text type="success">({t('qxmods.deps.installed')})</Text>
                        ) : !dep.download_url ? (
                          <Text type="danger">({t('qxmods.deps.unresolved')})</Text>
                        ) : null}
                      </Text>
                    </li>
                  );
                })}
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
                {optional.map((dep) => (
                  <li key={`${dep.source}:${dep.project_id}`} className="qxmods-deps-item">
                    <Checkbox
                      checked={optionalSelected.has(dep.project_id)}
                      disabled={!dep.download_url || installedProjectIds.has(`${dep.source}:${dep.project_id}`)}
                      onChange={(e) => {
                        setOptionalSelected((prev) => {
                          const next = new Set(prev);
                          if (e.target.checked) next.add(dep.project_id);
                          else next.delete(dep.project_id);
                          return next;
                        });
                      }}
                    >
                      {dep.project_name ?? dep.project_id}
                    </Checkbox>
                  </li>
                ))}
              </ul>
            </>
          ) : null}
        </>
      )}
    </Modal>
  );
}
