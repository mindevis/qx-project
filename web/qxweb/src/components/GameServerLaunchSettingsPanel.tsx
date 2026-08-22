import { useEffect, useState } from 'react';
import { Button, Form, Input, InputNumber, Typography } from 'antd';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  DEFAULT_GAME_SERVER_MEMORY_MB,
  defaultExtraJvmArgsForGameServer,
  updateVpsGameServer,
  type VpsGameServer,
} from '@/lib/vpsGameServers';

const { Paragraph, Title } = Typography;

type LaunchFormValues = {
  min_memory_mb: number;
  max_memory_mb: number;
  extra_jvm_args: string;
  extra_args: string;
};

type GameServerLaunchSettingsPanelProps = {
  vpsId: string;
  game: VpsGameServer;
  disabled?: boolean;
  onUpdated: (game: VpsGameServer) => void;
};

function splitLaunchArgs(raw: string): string[] {
  return raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}

export function GameServerLaunchSettingsPanel({
  vpsId,
  game,
  disabled,
  onUpdated,
}: GameServerLaunchSettingsPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [form] = Form.useForm<LaunchFormValues>();
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    form.setFieldsValue({
      min_memory_mb: game.min_memory_mb ?? DEFAULT_GAME_SERVER_MEMORY_MB,
      max_memory_mb: game.max_memory_mb ?? DEFAULT_GAME_SERVER_MEMORY_MB,
      extra_jvm_args: defaultExtraJvmArgsForGameServer(game).join('\n'),
      extra_args: (game.extra_args ?? []).join('\n'),
    });
  }, [form, game]);

  const save = async () => {
    let values: LaunchFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    if (values.min_memory_mb > values.max_memory_mb) {
      form.setFields([{ name: 'max_memory_mb', errors: [t('launcher.memoryRangeInvalid')] }]);
      return;
    }
    setSaving(true);
    try {
      const updated = await updateVpsGameServer(vpsId, game.id, {
        min_memory_mb: values.min_memory_mb,
        max_memory_mb: values.max_memory_mb,
        extra_jvm_args: splitLaunchArgs(values.extra_jvm_args),
        extra_args: splitLaunchArgs(values.extra_args),
      });
      onUpdated(updated);
      message.success(t('gameServerDetail.launchSaved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.launchSaveFailed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="game-server-launch-settings">
      <Title level={4} className="game-server-launch-title">
        {t('gameServerDetail.launchTitle')}
      </Title>
      <Paragraph type="secondary">{t('gameServerDetail.launchHint')}</Paragraph>
      <Form form={form} layout="vertical" disabled={disabled}>
        <Form.Item name="min_memory_mb" label={t('launcher.minMemoryMb')}>
          <InputNumber
            min={512}
            max={65536}
            step={512}
            addonAfter={t('common.megabytes')}
            style={{ width: '100%' }}
          />
        </Form.Item>
        <Form.Item
          name="max_memory_mb"
          label={t('launcher.maxMemoryMb')}
          dependencies={['min_memory_mb']}
          rules={[
            ({ getFieldValue }) => ({
              validator(_, value) {
                const min = getFieldValue('min_memory_mb') as number | undefined;
                if (value != null && min != null && value < min) {
                  return Promise.reject(new Error(t('launcher.memoryRangeInvalid')));
                }
                return Promise.resolve();
              },
            }),
          ]}
        >
          <InputNumber
            min={512}
            max={65536}
            step={512}
            addonAfter={t('common.megabytes')}
            style={{ width: '100%' }}
          />
        </Form.Item>
        <Form.Item
          name="extra_jvm_args"
          label={t('launcher.extraJvmArgs')}
          extra={t('gameServerDetail.extraJvmArgsHint')}
        >
          <Input.TextArea rows={10} placeholder="-XX:+UseG1GC" />
        </Form.Item>
        <Form.Item
          name="extra_args"
          label={t('gameServerDetail.extraArgs')}
          extra={t('gameServerDetail.extraArgsHint')}
        >
          <Input.TextArea rows={3} placeholder="--forceUpgrade" />
        </Form.Item>
        <Button type="primary" loading={saving} disabled={disabled} onClick={() => void save()}>
          {t('common.save')}
        </Button>
      </Form>
    </section>
  );
}
