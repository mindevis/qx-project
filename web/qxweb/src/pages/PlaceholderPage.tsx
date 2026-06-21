import { Typography } from 'antd';
import { useI18n } from '@/i18n/I18nContext';

type Props = {
  title: string;
  phase: string;
};

export function PlaceholderPage({ title, phase }: Props) {
  const { t } = useI18n();

  return (
    <div style={{ maxWidth: 560 }}>
      <Typography.Title level={3}>{title}</Typography.Title>
      <Typography.Paragraph type="secondary">
        {t('placeholder.body', { phase })}
      </Typography.Paragraph>
    </div>
  );
}
