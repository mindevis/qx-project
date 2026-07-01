import { useCallback, useEffect, useMemo, useState } from 'react';
import { Select, Spin, Typography } from 'antd';
import { api, type LauncherInstance, type MonitoringInstanceBinding } from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import './InstanceServerBinding.css';

const { Text } = Typography;

type Props = {
  gameServerId: string;
  mcVersion: string;
  loader: string;
  hasAddress: boolean;
};

export function GameServerInstanceBinding({
  gameServerId,
  mcVersion,
  loader,
  hasAddress,
}: Props) {
  const { t } = useI18n();
  const message = useMessage();
  const { isAuthenticated } = useAuth();
  const [instances, setInstances] = useState<LauncherInstance[]>([]);
  const [bindings, setBindings] = useState<MonitoringInstanceBinding[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!isAuthenticated) {
      setInstances([]);
      setBindings([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [instanceRes, bindingRes] = await Promise.all([
        api.listInstances(),
        api.listMonitoringBindings(),
      ]);
      setInstances(instanceRes.items ?? []);
      setBindings(bindingRes.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
      setInstances([]);
      setBindings([]);
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const compatibleInstances = useMemo(
    () =>
      instances.filter(
        (item) => item.mc_version === mcVersion && item.loader === loader,
      ),
    [instances, loader, mcVersion],
  );

  const boundInstanceId = useMemo(
    () => bindings.find((b) => b.game_server_id === gameServerId)?.instance_id,
    [bindings, gameServerId],
  );

  const options = useMemo(
    () =>
      compatibleInstances.map((instance) => ({
        value: instance.id,
        label: instance.name,
      })),
    [compatibleInstances],
  );

  const handleChange = async (instanceId: string | null) => {
    setSaving(true);
    try {
      if (instanceId) {
        await api.setMonitoringBinding(gameServerId, instanceId);
        message.success(t('monitoring.bindingSaved'));
      } else if (boundInstanceId) {
        await api.clearMonitoringBinding(gameServerId);
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
      <div className="game-server-binding">
        <Text type="secondary">{t('qxmods.binding.signIn')}</Text>
      </div>
    );
  }

  return (
    <div className="game-server-binding">
      <Text strong className="game-server-binding-title">
        {t('gameServerDetail.bindingTitle')}
      </Text>
      <Text type="secondary" className="game-server-binding-hint">
        {t('gameServerDetail.bindingHint')}
      </Text>
      {!hasAddress ? (
        <Text type="warning">{t('gameServerDetail.bindingNeedsAddress')}</Text>
      ) : loading ? (
        <Spin size="small" />
      ) : options.length === 0 ? (
        <Text type="secondary">{t('gameServerDetail.bindingNoInstances', { mc: mcVersion, loader })}</Text>
      ) : (
        <Select
          allowClear
          showSearch
          placeholder={t('monitoring.bindInstancePlaceholder')}
          className="game-server-binding-select"
          loading={saving}
          disabled={saving}
          value={boundInstanceId ?? undefined}
          options={options}
          onChange={(value) => void handleChange(value ?? null)}
          optionFilterProp="label"
        />
      )}
    </div>
  );
}
