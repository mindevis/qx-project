import { useCallback, useEffect, useState } from 'react';
import { Button, Input, InputNumber, Space, Spin, Switch, Typography } from 'antd';
import { api, type GameServerProperty } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

type GameServerPropertiesPanelProps = {
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
};

export function GameServerPropertiesPanel({
  vpsId,
  gameServerId,
  agentOnline,
}: GameServerPropertiesPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [properties, setProperties] = useState<GameServerProperty[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingKey, setSavingKey] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!agentOnline) {
      setProperties([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await api.getVpsGameServerProperties(vpsId, gameServerId);
      setProperties(res.properties ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.propertiesLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [agentOnline, gameServerId, message, t, vpsId]);

  useEffect(() => {
    void load();
  }, [load]);

  const save = async (key: string, value: string) => {
    setSavingKey(key);
    try {
      await api.patchVpsGameServerProperties(vpsId, gameServerId, { [key]: value });
      setProperties((prev) =>
        prev.map((item) => (item.key === key ? { ...item, value } : item)),
      );
      message.success(t('gameServerDetail.propertySaved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
      await load();
    } finally {
      setSavingKey(null);
    }
  };

  if (!agentOnline) {
    return (
      <Typography.Paragraph type="secondary">
        {t('servers.gameServersAgentRequired')}
      </Typography.Paragraph>
    );
  }

  if (loading) {
    return (
      <div className="servers-loading">
        <Spin />
      </div>
    );
  }

  if (properties.length === 0) {
    return <Typography.Paragraph type="secondary">{t('gameServerDetail.propertiesEmpty')}</Typography.Paragraph>;
  }

  return (
    <div className="game-server-properties">
      {properties.map((item) => (
        <div key={item.key} className="game-server-property-row">
          <label className="game-server-property-label" htmlFor={`prop-${item.key}`}>
            {item.key}
          </label>
          <div className="game-server-property-control">
            {item.boolean ? (
              <Switch
                id={`prop-${item.key}`}
                checked={item.value.toLowerCase() === 'true'}
                loading={savingKey === item.key}
                onChange={(checked) => void save(item.key, checked ? 'true' : 'false')}
              />
            ) : /^\d+$/.test(item.value) ? (
              <InputNumber
                id={`prop-${item.key}`}
                value={Number(item.value)}
                disabled={savingKey === item.key}
                onChange={(value) => {
                  if (value != null) {
                    void save(item.key, String(value));
                  }
                }}
              />
            ) : (
              <Space.Compact className="game-server-property-text">
                <Input
                  id={`prop-${item.key}`}
                  defaultValue={item.value}
                  disabled={savingKey === item.key}
                  onPressEnter={(e) => {
                    const target = e.target as HTMLInputElement;
                    if (target.value !== item.value) {
                      void save(item.key, target.value);
                    }
                  }}
                />
                <Button
                  loading={savingKey === item.key}
                  onClick={(e) => {
                    const row = (e.currentTarget as HTMLElement).closest('.game-server-property-row');
                    const input = row?.querySelector('input') as HTMLInputElement | null;
                    if (input && input.value !== item.value) {
                      void save(item.key, input.value);
                    }
                  }}
                >
                  {t('common.save')}
                </Button>
              </Space.Compact>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
