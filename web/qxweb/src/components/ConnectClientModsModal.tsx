import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Checkbox, Modal, Spin, Typography } from 'antd';
import { api, type ConnectModStatus } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { modalMotionProps } from '@/lib/modal';

const { Text, Paragraph, Title } = Typography;

type ConnectClientModsModalProps = {
  open: boolean;
  gameServerId: string;
  instanceId: string;
  serverName: string;
  onClose: () => void;
  onConfirm: () => void;
};

type ClientContentSection = {
  key: 'mod' | 'resourcepack' | 'shader';
  items: ConnectModStatus['client_mods'];
  allInstalled: boolean;
  savedEnabled?: string[];
  titleKey: string;
  emptyKey: string;
  allInstalledKey: string;
};

function buildInitialSelection(
  items: ConnectModStatus['client_mods'],
  savedEnabled: string[] | undefined,
  prefix: string,
): Set<string> {
  const saved = new Set((savedEnabled ?? []).map((name) => name.toLowerCase()));
  const initial = new Set<string>();
  for (const item of items ?? []) {
    const key = `${prefix}:${item.filename.toLowerCase()}`;
    if (savedEnabled != null) {
      if (saved.has(item.filename.toLowerCase())) initial.add(key);
    } else {
      initial.add(key);
    }
  }
  return initial;
}

export function ConnectClientModsModal({
  open,
  gameServerId,
  instanceId,
  serverName,
  onClose,
  onConfirm,
}: ConnectClientModsModalProps) {
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<ConnectModStatus | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    void (async () => {
      try {
        const res = await api.getConnectModStatus(gameServerId, instanceId);
        if (cancelled) return;
        setStatus(res);
        const initial = new Set<string>();
        for (const [prefix, items, saved] of [
          ['mod', res.client_mods, res.saved_client_mod_enabled] as const,
          ['resourcepack', res.client_resourcepacks, res.saved_client_resourcepack_enabled] as const,
          ['shader', res.client_shaders, res.saved_client_shader_enabled] as const,
        ]) {
          for (const key of buildInitialSelection(items, saved, prefix)) {
            initial.add(key);
          }
        }
        setSelected(initial);
      } catch {
        if (!cancelled) setStatus(null);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [gameServerId, instanceId, open]);

  const sections = useMemo<ClientContentSection[]>(
    () =>
      status
        ? [
            {
              key: 'mod',
              items: status.client_mods,
              allInstalled: status.all_client_mods_installed,
              savedEnabled: status.saved_client_mod_enabled,
              titleKey: 'monitoring.connectMods.clientModsTitle',
              emptyKey: 'monitoring.connectMods.noClientMods',
              allInstalledKey: 'monitoring.connectMods.allInstalled',
            },
            {
              key: 'resourcepack',
              items: status.client_resourcepacks,
              allInstalled: status.all_client_resourcepacks_installed,
              savedEnabled: status.saved_client_resourcepack_enabled,
              titleKey: 'monitoring.connectMods.clientResourcepacksTitle',
              emptyKey: 'monitoring.connectMods.noClientResourcepacks',
              allInstalledKey: 'monitoring.connectMods.allResourcepacksInstalled',
            },
            {
              key: 'shader',
              items: status.client_shaders,
              allInstalled: status.all_client_shaders_installed,
              savedEnabled: status.saved_client_shader_enabled,
              titleKey: 'monitoring.connectMods.clientShadersTitle',
              emptyKey: 'monitoring.connectMods.noClientShaders',
              allInstalledKey: 'monitoring.connectMods.allShadersInstalled',
            },
          ]
        : [],
    [status],
  );

  const selectedByType = useMemo(() => {
    const mods: string[] = [];
    const resourcepacks: string[] = [];
    const shaders: string[] = [];
    for (const section of sections) {
      for (const item of section.items ?? []) {
        const key = `${section.key}:${item.filename.toLowerCase()}`;
        if (!selected.has(key)) continue;
        if (section.key === 'mod') mods.push(item.filename);
        if (section.key === 'resourcepack') resourcepacks.push(item.filename);
        if (section.key === 'shader') shaders.push(item.filename);
      }
    }
    return { mods, resourcepacks, shaders };
  }, [sections, selected]);

  const hasClientSelection = sections.some((section) => (section.items?.length ?? 0) > 0);

  const handleConfirm = async () => {
    setSaving(true);
    try {
      if (hasClientSelection) {
        await api.setClientModPrefs(gameServerId, {
          enabled_filenames: selectedByType.mods,
          enabled_resourcepack_filenames: selectedByType.resourcepacks,
          enabled_shader_filenames: selectedByType.shaders,
        });
      }
      onConfirm();
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      {...modalMotionProps}
      title={t('monitoring.connectMods.title', { server: serverName })}
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose}>
          {t('common.cancel')}
        </Button>,
        <Button key="connect" type="primary" loading={saving} onClick={() => void handleConfirm()}>
          {t('monitoring.connectMods.confirm')}
        </Button>,
      ]}
    >
      {loading ? (
        <Spin />
      ) : !status ? (
        <Alert type="warning" showIcon message={t('monitoring.connectMods.loadFailed')} />
      ) : (
        <>
          <Paragraph type="secondary">{t('monitoring.connectMods.hint')}</Paragraph>
          {status.server_mod_count > 0 ? (
            <Text type="secondary" className="monitoring-connect-mods-server-info">
              {t('monitoring.connectMods.serverModsInfo', { count: status.server_mod_count })}
            </Text>
          ) : null}
          {status.server_resourcepack_count > 0 ? (
            <Text type="secondary" className="monitoring-connect-mods-server-info">
              {t('monitoring.connectMods.serverResourcepacksInfo', {
                count: status.server_resourcepack_count,
              })}
            </Text>
          ) : null}
          {status.server_shader_count > 0 ? (
            <Text type="secondary" className="monitoring-connect-mods-server-info">
              {t('monitoring.connectMods.serverShadersInfo', { count: status.server_shader_count })}
            </Text>
          ) : null}
          {sections.map((section) => (
            <section key={section.key} className="monitoring-connect-mods-section">
              <Title level={5}>{t(section.titleKey)}</Title>
              {(section.items?.length ?? 0) === 0 ? (
                <Alert type="info" showIcon message={t(section.emptyKey)} />
              ) : (
                <ul className="monitoring-connect-mods-list">
                  {section.items.map((item) => {
                    const key = `${section.key}:${item.filename.toLowerCase()}`;
                    return (
                      <li key={key}>
                        <Checkbox
                          checked={selected.has(key)}
                          onChange={(e) => {
                            setSelected((prev) => {
                              const next = new Set(prev);
                              if (e.target.checked) next.add(key);
                              else next.delete(key);
                              return next;
                            });
                          }}
                        >
                          <span>{item.filename}</span>
                          {!item.installed_locally ? (
                            <Text type="warning" className="monitoring-connect-mods-missing">
                              {' '}
                              · {t('monitoring.connectMods.notInstalled')}
                            </Text>
                          ) : null}
                        </Checkbox>
                      </li>
                    );
                  })}
                </ul>
              )}
              {section.allInstalled && (section.items?.length ?? 0) > 0 ? (
                <Alert type="success" showIcon message={t(section.allInstalledKey)} />
              ) : null}
            </section>
          ))}
        </>
      )}
    </Modal>
  );
}
