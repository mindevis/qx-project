import { Tag } from 'antd';
import { useI18n } from '@/i18n/I18nContext';
import type { ModVersionChannel } from '@/lib/modVersionChannel';

const TAG_COLOR: Record<ModVersionChannel, string> = {
  release: 'green',
  beta: 'gold',
  alpha: 'magenta',
};

type ModVersionChannelBadgeProps = {
  channel: ModVersionChannel;
  className?: string;
};

export function ModVersionChannelBadge({ channel, className }: ModVersionChannelBadgeProps) {
  const { t } = useI18n();
  return (
    <Tag color={TAG_COLOR[channel]} className={className} variant="filled">
      {t(`qxmods.versionChannel.${channel}`)}
    </Tag>
  );
}
