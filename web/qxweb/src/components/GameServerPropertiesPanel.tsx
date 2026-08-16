import { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Form, Input, InputNumber, Space, Spin, Switch } from 'antd';
import { api, type GameServerProperty } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { getServerPropertyMeta } from '@/lib/serverPropertyHints';

type GameServerPropertiesPanelProps = {
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
};

function fieldValue(item: GameServerProperty): boolean | number | string {
  if (item.boolean) {
    return item.value.toLowerCase() === 'true';
  }
  if (/^\d+$/.test(item.value)) {
    return Number(item.value);
  }
  return item.value;
}

export function GameServerPropertiesPanel({
  vpsId,
  gameServerId,
  agentOnline,
}: GameServerPropertiesPanelProps) {
  const { locale, t } = useI18n();
  const message = useMessage();
  const [form] = Form.useForm<Record<string, boolean | number | string>>();
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
      const next = res.properties ?? [];
      setProperties(next);
      form.setFieldsValue(Object.fromEntries(next.map((item) => [item.key, fieldValue(item)])));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.propertiesLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [agentOnline, form, gameServerId, message, t, vpsId]);

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

  const saveTextField = (key: string, current: string) => {
    const value = String(form.getFieldValue(key) ?? '');
    if (value !== current) {
      void save(key, value);
    }
  };

  if (!agentOnline) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('servers.gameServersAgentRequired')}
      />
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
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('gameServerDetail.propertiesEmpty')}
      />
    );
  }

  return (
    <Form form={form} component="div" className="game-server-properties">
      {properties.map((item) => {
        const meta = getServerPropertyMeta(locale, item.key);
        return (
          <div key={item.key} className="game-server-property-row">
            <label className="game-server-property-label" htmlFor={`prop-${item.key}`}>
              <span className="game-server-property-title">{meta.title}</span>
              {meta.title !== item.key ? (
                <span className="game-server-property-key">{item.key}</span>
              ) : null}
              {meta.description ? (
                <span className="game-server-property-desc">{meta.description}</span>
              ) : null}
            </label>
            <div className="game-server-property-control">
              {item.boolean ? (
                <Form.Item name={item.key} noStyle valuePropName="checked">
                  <Switch
                    id={`prop-${item.key}`}
                    loading={savingKey === item.key}
                    onChange={(checked) => void save(item.key, checked ? 'true' : 'false')}
                  />
                </Form.Item>
              ) : /^\d+$/.test(item.value) ? (
                <Form.Item name={item.key} noStyle>
                  <InputNumber
                    id={`prop-${item.key}`}
                    disabled={savingKey === item.key}
                    onChange={(value) => {
                      if (value != null) {
                        void save(item.key, String(value));
                      }
                    }}
                  />
                </Form.Item>
              ) : (
                <Space.Compact className="game-server-property-text">
                  <Form.Item name={item.key} noStyle>
                    <Input
                      id={`prop-${item.key}`}
                      disabled={savingKey === item.key}
                      onPressEnter={() => saveTextField(item.key, item.value)}
                    />
                  </Form.Item>
                  <Button
                    loading={savingKey === item.key}
                    onClick={() => saveTextField(item.key, item.value)}
                  >
                    {t('common.save')}
                  </Button>
                </Space.Compact>
              )}
            </div>
          </div>
        );
      })}
    </Form>
  );
}
