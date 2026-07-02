import { Tag } from 'antd';
import type { ModCatalogItem } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { modSyncSide } from '@/lib/modSync';

type ModSideBadgeProps = {
  item: Pick<ModCatalogItem, 'client_side' | 'server_side'>;
  className?: string;
};

export function ModSideBadge({ item, className }: ModSideBadgeProps) {
  const { t } = useI18n();
  const side = modSyncSide(item);
  if (side === 'unknown') return null;

  const color = side === 'client' ? 'blue' : side === 'server' ? 'volcano' : 'purple';
  return (
    <Tag color={color} className={className}>
      {t(`qxmods.side.${side}`)}
    </Tag>
  );
}
