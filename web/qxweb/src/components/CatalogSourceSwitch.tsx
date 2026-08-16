import { Button, Segmented } from 'antd';
import { LinkOutlined } from '@ant-design/icons';
import type { ModCatalogItem, ModSource } from '@/api/client';
import { ModSourceBadge } from '@/components/ModSourceBadge';
import { useI18n } from '@/i18n/I18nContext';

type CatalogSourceSwitchProps = {
  items: ModCatalogItem[];
  value: ModSource;
  onChange: (source: ModSource) => void;
};

export function CatalogSourceSwitch({ items, value, onChange }: CatalogSourceSwitchProps) {
  const { t } = useI18n();
  const sources = items.filter((item) => item.source === 'modrinth' || item.source === 'curseforge');
  if (sources.length < 2) {
    return <ModSourceBadge source={items[0]?.source ?? value} />;
  }

  return (
    <Segmented<ModSource>
      className="qxmods-source-switch"
      size="small"
      value={value}
      aria-label={t('qxmods.filters.sourcePick')}
      onChange={onChange}
      options={sources.map((item) => ({
        value: item.source,
        label: t(`qxmods.source.${item.source}`),
      }))}
    />
  );
}

export function CatalogSourceLinks({ items }: { items: ModCatalogItem[] }) {
  const { t } = useI18n();
  const links = items.filter((item) => item.external_url);
  if (links.length === 0) {
    return null;
  }
  return (
    <div className="qxmods-source-links">
      {links.map((item) => {
        const label = t('qxmods.viewOnNamed', { source: t(`qxmods.source.${item.source}`) });
        return (
          <Button
            key={`${item.source}:${item.id}`}
            type="link"
            href={item.external_url}
            target="_blank"
            rel="noreferrer"
            icon={<LinkOutlined aria-hidden />}
            aria-label={label}
          >
            {label}
          </Button>
        );
      })}
    </div>
  );
}
