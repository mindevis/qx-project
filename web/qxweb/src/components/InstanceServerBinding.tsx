import { useCallback, useEffect, useMemo, useState } from 'react';
import { QuestionCircleOutlined } from '@ant-design/icons';
import { Select, Spin, Tooltip, Typography } from 'antd';
import {
  api,
  type LauncherInstance,
  type MonitoringInstanceBinding,
  type MonitoringServer,
} from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import './InstanceServerBinding.css';

const { Text } = Typography;

type InstanceServerBindingProps = {
  instance: LauncherInstance;
  variant?: 'panel' | 'card';
};

export function InstanceServerBinding({ instance, variant = 'panel' }: InstanceServerBindingProps) {
  const { t } = useI18n();
  const message = useMessage();
  const { isAuthenticated } = useAuth();
  const [servers, setServers] = useState<MonitoringServer[]>([]);
  const [bindings, setBindings] = useState<MonitoringInstanceBinding[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const isCard = variant === 'card';

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
        api.listBindableServers({
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
      <div className={`qxmods-binding-panel${isCard ? ' qxmods-binding-panel--card' : ''}`}>
        <Text type="secondary">{t('qxmods.binding.signIn')}</Text>
      </div>
    );
  }

  const title = (
    <span className="qxmods-binding-title-row">
      <Text strong={!isCard} className="qxmods-binding-title">
        {t('qxmods.binding.title')}
      </Text>
      {isCard ? (
        <Tooltip title={t('qxmods.binding.hint')}>
          <QuestionCircleOutlined className="qxmods-binding-help" aria-label={t('qxmods.binding.hint')} />
        </Tooltip>
      ) : null}
    </span>
  );

  return (
    <div className={`qxmods-binding-panel${isCard ? ' qxmods-binding-panel--card' : ''}`}>
      {title}
      {!isCard ? (
        <Text type="secondary" className="qxmods-binding-hint">
          {t('qxmods.binding.hint')}
        </Text>
      ) : null}
      {loading ? (
        <Spin size="small" />
      ) : options.length === 0 ? (
        <Text type="secondary" className="qxmods-binding-empty">
          {t('qxmods.binding.empty', {
            mc: instance.mc_version,
            loader: instance.loader,
          })}
        </Text>
      ) : (
        <Select
          allowClear
          showSearch
          placeholder={t('qxmods.binding.placeholder')}
          className={`qxmods-binding-select${isCard ? ' qxmods-binding-select--card' : ''}`}
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
