import { useCallback, useEffect, useMemo, useState } from 'react';
import { Spin, Typography } from 'antd';
import { api, type MonitoringInstanceBinding } from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import './InstanceServerBinding.css';

const { Text } = Typography;

type Props = {
  gameServerId: string;
  hasAddress: boolean;
};

export function GameServerInstanceBinding({ gameServerId, hasAddress }: Props) {
  const { t } = useI18n();
  const message = useMessage();
  const { isAuthenticated } = useAuth();
  const [bindings, setBindings] = useState<MonitoringInstanceBinding[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!isAuthenticated) {
      setBindings([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const bindingRes = await api.listMonitoringBindings();
      setBindings(bindingRes.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
      setBindings([]);
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const bound = useMemo(
    () => bindings.find((b) => b.game_server_id === gameServerId),
    [bindings, gameServerId],
  );

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
        {t('gameServerDetail.bindingLockedHint')}
      </Text>
      {!hasAddress ? (
        <Text type="warning">{t('gameServerDetail.bindingNeedsAddress')}</Text>
      ) : loading ? (
        <Spin size="small" />
      ) : bound ? (
        <Text>
          {t('gameServerDetail.bindingLockedValue', {
            name: bound.instance_name || bound.instance_id,
          })}
        </Text>
      ) : (
        <Text type="secondary">{t('gameServerDetail.bindingLockedEmpty')}</Text>
      )}
    </div>
  );
}
