import { useCallback, useEffect, useMemo, useState } from 'react';
import { QuestionCircleOutlined } from '@ant-design/icons';
import { Spin, Tooltip, Typography } from 'antd';
import {
  api,
  type LauncherInstance,
  type MonitoringInstanceBinding,
  type MonitoringServer,
} from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { isServerManagedInstance } from '@/lib/serverManagedInstance';
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
  const isCard = variant === 'card';
  const managed = isServerManagedInstance(instance);

  const load = useCallback(async () => {
    if (!isAuthenticated) {
      setServers([]);
      setBindings([]);
      setLoading(false);
      return;
    }
    if (!managed) {
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
  }, [instance.loader, instance.mc_version, isAuthenticated, managed, message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const boundServerId = useMemo(
    () =>
      instance.managed_by_game_server_id ??
      bindings.find((b) => b.instance_id === instance.id)?.game_server_id,
    [bindings, instance.id, instance.managed_by_game_server_id],
  );

  const boundServer = useMemo(
    () => servers.find((server) => server.id === boundServerId),
    [boundServerId, servers],
  );

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
        <Tooltip title={t('qxmods.binding.lockedHint')}>
          <QuestionCircleOutlined className="qxmods-binding-help" aria-label={t('qxmods.binding.lockedHint')} />
        </Tooltip>
      ) : null}
    </span>
  );

  const lockedBody = () => {
    if (loading) return <Spin size="small" />;
    if (boundServer) {
      return (
        <Text>
          {t('qxmods.binding.lockedValue', {
            name: `${boundServer.name} (${boundServer.address}:${boundServer.port})`,
          })}
        </Text>
      );
    }
    if (managed) {
      return <Text type="secondary">{t('qxmods.binding.lockedUnmanaged')}</Text>;
    }
    if (isCard) {
      return <Text type="secondary">{t('qxmods.binding.personalOnly')}</Text>;
    }
    return null;
  };

  return (
    <div className={`qxmods-binding-panel${isCard ? ' qxmods-binding-panel--card' : ''}`}>
      {title}
      {!isCard ? (
        <Text type="secondary" className="qxmods-binding-hint">
          {managed ? t('qxmods.binding.lockedHint') : t('qxmods.binding.personalOnly')}
        </Text>
      ) : null}
      {lockedBody()}
    </div>
  );
}
