import { useCallback, useEffect, useMemo, useState } from 'react';
import { Select, Spin, Typography } from 'antd';
import {
  api,
  type MonitoringInstanceBinding,
  type MonitoringServer,
} from '@/api/client';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

const { Text } = Typography;

export function InstanceServerBinding() {
  const { t } = useI18n();
  const message = useMessage();
  const { isAuthenticated } = useAuth();
  const { instance } = useInstanceMods();
  const [servers, setServers] = useState<MonitoringServer[]>([]);
  const [bindings, setBindings] = useState<MonitoringInstanceBinding[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!isAuthenticated) {
      setServers([]);
      setBindings([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [serverRes, bindingRes] = await Promise.all([
        api.listMonitoringServers({
          mc_version: instance.mc_version,
          loader: instance.loader,
        }),
        api.listMonitoringBindings(),
      ]);
      setServers(serverRes.items ?? []);
      setBindings(bindingRes.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.binding.loadFailed'));
      setServers([]);
      setBindings([]);
    } finally {
      setLoading(false);
    }
  }, [instance.loader, instance.mc_version, isAuthenticated, message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const boundServerId = useMemo(
    () => bindings.find((b) => b.instance_id === instance.id)?.game_server_id,
    [bindings, instance.id],
  );

  const options = useMemo(
    () =>
      servers.map((server) => ({
        value: server.id,
        label: `${server.name} (${server.address}:${server.port})`,
      })),
    [servers],
  );

  const handleChange = async (serverId: string | null) => {
    if (!isAuthenticated) return;
    setSaving(true);
    try {
      if (boundServerId && boundServerId !== serverId) {
        await api.clearMonitoringBinding(boundServerId);
      }
      if (serverId) {
        await api.setMonitoringBinding(serverId, instance.id);
        message.success(t('monitoring.bindingSaved'));
      } else if (boundServerId) {
        await api.clearMonitoringBinding(boundServerId);
        message.success(t('monitoring.bindingCleared'));
      }
      await load();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  if (!isAuthenticated) {
    return (
      <div className="qxmods-binding-panel">
        <Text type="secondary">{t('qxmods.binding.signIn')}</Text>
      </div>
    );
  }

  return (
    <div className="qxmods-binding-panel">
      <Text strong className="qxmods-binding-title">
        {t('qxmods.binding.title')}
      </Text>
      <Text type="secondary" className="qxmods-binding-hint">
        {t('qxmods.binding.hint')}
      </Text>
      {loading ? (
        <Spin size="small" />
      ) : (
        <Select
          allowClear
          showSearch
          placeholder={t('qxmods.binding.placeholder')}
          className="qxmods-binding-select"
          loading={saving}
          disabled={saving}
          value={boundServerId ?? undefined}
          options={options}
          onChange={(value) => void handleChange(value ?? null)}
          optionFilterProp="label"
        />
      )}
    </div>
  );
}
