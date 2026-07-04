import { Tag } from 'antd';
import type { InstanceResource } from '@/api/client';

type ResourceMetaBadgesProps = {
  item: InstanceResource;
  className?: string;
};

export function ResourceMetaBadges({ item, className }: ResourceMetaBadgesProps) {
  const classNames = ['launcher-resource-card-badges', className].filter(Boolean).join(' ');

  return (
    <div className={classNames}>
      {item.version_number ? (
        <Tag bordered={false} className="launcher-resource-meta-tag launcher-resource-meta-tag--version">
          {item.version_number}
        </Tag>
      ) : null}
    </div>
  );
}
