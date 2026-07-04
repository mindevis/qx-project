import { Tag } from 'antd';
import type { InstanceResource } from '@/api/client';
import { formatDownloadCount, formatFileSize } from '@/lib/formatFileSize';

type ResourceMetaBadgesProps = {
  item: InstanceResource;
  t: (key: string) => string;
  className?: string;
};

export function ResourceMetaBadges({ item, t, className }: ResourceMetaBadgesProps) {
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
