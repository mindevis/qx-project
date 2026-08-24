import { Tag } from 'antd';
import type { InstanceResource } from '@/api/client';
import { ModVersionChannelBadge } from '@/components/ModVersionChannelBadge';
import { resolveModVersionChannel } from '@/lib/modVersionChannel';

type ResourceMetaBadgesProps = {
  item: InstanceResource;
  className?: string;
};

export function ResourceMetaBadges({ item, className }: ResourceMetaBadgesProps) {
  const classNames = ['launcher-resource-card-badges', className].filter(Boolean).join(' ');

  return (
    <div className={classNames}>
      {item.version_number ? (
        <Tag variant="filled" className="launcher-resource-meta-tag launcher-resource-meta-tag--version">
          {item.version_number}
        </Tag>
      ) : null}
      <ModVersionChannelBadge
        channel={resolveModVersionChannel(item.version_type, item.version_number, item.filename)}
      />
    </div>
  );
}
