import { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Input, Spin, Typography } from 'antd';
import { api } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

const OPTIONS_PATH = 'options.txt';

type InstanceOptionsPanelProps = {
  instanceId: string;
  deviceLinked: boolean;
};

export function InstanceOptionsPanel({ instanceId, deviceLinked }: InstanceOptionsPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const loadOptions = useCallback(async () => {
    if (!deviceLinked) {
      setContent('');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await api.readInstanceFile(instanceId, OPTIONS_PATH);
      setContent(res.content);
    } catch {
      setContent('');
    } finally {
      setLoading(false);
    }
  }, [deviceLinked, instanceId]);

  useEffect(() => {
    void loadOptions();
  }, [loadOptions]);

  const saveOptions = async () => {
    setSaving(true);
    try {
      await api.writeInstanceFile(instanceId, OPTIONS_PATH, content);
      message.success(t('gameServerDetail.fileSaved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  if (!deviceLinked) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('launcher.instanceSettingsModsConfigNote')}
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

  return (
    <div className="game-server-files-editor">
      <Typography.Paragraph type="secondary">{t('launcher.instanceOptionsHint')}</Typography.Paragraph>
      <Input.TextArea
        className="game-server-files-textarea"
        value={content}
        rows={18}
        onChange={(e) => setContent(e.target.value)}
      />
      <Button type="primary" loading={saving} onClick={() => void saveOptions()} style={{ marginTop: 12 }}>
        {t('common.save')}
      </Button>
    </div>
  );
}
