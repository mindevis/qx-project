import { Select } from 'antd';
import type { ModVersion } from '@/api/client';
import { ModVersionChannelBadge } from '@/components/ModVersionChannelBadge';
import { resolveModVersionChannel } from '@/lib/modVersionChannel';

export function ModVersionSelect({
  versions,
  value,
  loading,
  disabled,
  className,
  placeholder,
  ariaLabel,
  size,
  onChange,
  onOpenChange,
}: {
  versions: ModVersion[];
  value?: string;
  loading?: boolean;
  disabled?: boolean;
  className?: string;
  placeholder?: string;
  ariaLabel?: string;
  size?: 'small' | 'middle' | 'large';
  onChange: (versionId: string) => void;
  onOpenChange?: (open: boolean) => void;
}) {
  const selected = versions.find((item) => item.id === value) ?? versions[0];
  return (
    <Select
      showSearch
      size={size}
      optionFilterProp="label"
      placeholder={placeholder}
      className={className ?? 'qxmods-install-version-select'}
      loading={loading}
      disabled={disabled}
      value={selected?.id}
      options={versions.map((version) => ({
        value: version.id,
        label: version.version_number,
        version,
      }))}
      aria-label={ariaLabel}
      optionRender={(option) => {
        const version = option.data.version as ModVersion | undefined;
        const channel = resolveModVersionChannel(
          version?.version_type,
          version?.version_number,
          String(option.label ?? ''),
        );
        return (
          <span className="qxmods-version-option">
            <span className="qxmods-version-option-name">{option.label}</span>
            <ModVersionChannelBadge channel={channel} />
          </span>
        );
      }}
      labelRender={(props) => {
        const version =
          versions.find((item) => item.id === props.value) ??
          (selected?.id === props.value ? selected : undefined);
        const channel = resolveModVersionChannel(
          version?.version_type,
          version?.version_number,
          String(props.label ?? ''),
        );
        return (
          <span className="qxmods-version-option qxmods-version-option--selected">
            <span className="qxmods-version-option-name">{props.label}</span>
            <ModVersionChannelBadge channel={channel} />
          </span>
        );
      }}
      onChange={onChange}
      onOpenChange={onOpenChange}
    />
  );
}
