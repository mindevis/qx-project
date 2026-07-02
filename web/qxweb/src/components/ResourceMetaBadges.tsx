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
      <Tag bordered={false} className="launcher-resource-meta-tag launcher-resource-meta-tag--type">
        {t(`qxmods.tabs.${item.resource_type}`)}
      </Tag>
      {item.version_number ? (
        <Tag bordered={false} className="launcher-resource-meta-tag launcher-resource-meta-tag--version">
          {item.version_number}
        </Tag>
      ) : null}
      {item.filename ? (
        <Tag bordered={false} className="launcher-resource-meta-tag launcher-resource-meta-tag--file">
          {item.filename}
        </Tag>
      ) : null}
      {item.file_size ? (
        <Tag bordered={false} className="launcher-resource-meta-tag launcher-resource-meta-tag--size">
          {formatFileSize(item.file_size)}
        </Tag>
      ) : null}
      {item.downloads ? (
        <Tag bordered={false} className="launcher-resource-meta-tag launcher-resource-meta-tag--downloads">
          {formatDownloadCount(item.downloads)} {t('qxmods.installed.downloads')}
        </Tag>
      ) : null}
    </div>
  );
}
