import { Tag } from 'antd';
import type { ModSource } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';

type ModSourceBadgeProps = {
  source: ModSource;
  className?: string;
};

export function ModSourceBadge({ source, className }: ModSourceBadgeProps) {
  const { t } = useI18n();
  const color =
    source === 'curseforge' ? 'orange' : source === 'upload' ? 'blue' : 'green';
  return (
    <Tag color={color} className={className}>
      {t(`qxmods.source.${source}`)}
    </Tag>
  );
}
