import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Empty, Input, Select, Spin, Typography } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { api, type SkinCatalogEntry } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import './SkinCatalogPanel.css';

const { Text } = Typography;

type Props = {
  onApplied?: () => void;
};

const CATEGORIES = ['', 'popular', 'creators', 'classic'] as const;

export function SkinCatalogPanel({ onApplied }: Props) {
  const { t } = useI18n();
  const message = useMessage();
  const [loading, setLoading] = useState(true);
  const [applyingId, setApplyingId] = useState<string>();
  const [category, setCategory] = useState('');
  const [filterQuery, setFilterQuery] = useState('');
  const [filterDraft, setFilterDraft] = useState('');
  const [usernameDraft, setUsernameDraft] = useState('');
  const [items, setItems] = useState<SkinCatalogEntry[]>([]);

  const loadCatalog = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listSkinCatalog(category ? { category } : undefined);
      setItems(res.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('skins.catalog.loadFailed'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [category, message, t]);

  useEffect(() => {
    void loadCatalog();
  }, [loadCatalog]);

  useEffect(() => {
    const id = window.setTimeout(() => setFilterQuery(filterDraft.trim()), 400);
    return () => window.clearTimeout(id);
  }, [filterDraft]);

  const filtered = useMemo(() => {
    const q = filterQuery.toLowerCase();
    if (!q) return items;
    return items.filter(
      (item) =>
        item.name.toLowerCase().includes(q) || item.username.toLowerCase().includes(q),
    );
  }, [items, filterQuery]);

  const applyCatalog = async (entry: SkinCatalogEntry) => {
    setApplyingId(entry.id);
    try {
      await api.applyCosmeticsSkin({ catalog_id: entry.id });
      message.success(t('skins.catalog.applied'));
      onApplied?.();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('skins.catalog.applyFailed'));
    } finally {
      setApplyingId(undefined);
    }
  };

  const applyUsername = async () => {
    const username = usernameDraft.trim();
    if (!username) return;
    setApplyingId('search');
    try {
      await api.applyCosmeticsSkin({ username });
      message.success(t('skins.catalog.applied'));
      onApplied?.();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('skins.catalog.applyFailed'));
    } finally {
      setApplyingId(undefined);
    }
  };

  const categoryOptions = CATEGORIES.map((value) => ({
    value,
    label: value ? t(`skins.catalog.categories.${value}`) : t('skins.catalog.categories.all'),
  }));

  return (
    <section className="skins-catalog">
      <div className="skins-catalog-toolbar">
        <Select
          value={category}
          onChange={setCategory}
          options={categoryOptions}
          className="skins-catalog-category"
        />
        <Input
          allowClear
          value={filterDraft}
          onChange={(e) => setFilterDraft(e.target.value)}
          placeholder={t('skins.catalog.filterPlaceholder')}
          className="skins-catalog-search"
          prefix={<SearchOutlined aria-hidden />}
          aria-label={t('skins.catalog.search')}
        />
        <Input
          allowClear
          value={usernameDraft}
          onChange={(e) => setUsernameDraft(e.target.value)}
          onPressEnter={() => void applyUsername()}
          placeholder={t('skins.catalog.usernamePlaceholder')}
          className="skins-catalog-username"
        />
        <Button
          type="primary"
          loading={applyingId === 'search'}
          disabled={!usernameDraft.trim()}
          onClick={() => void applyUsername()}
        >
          {t('skins.catalog.applyUsername')}
        </Button>
      </div>

      {loading ? (
        <div className="skins-catalog-loading">
          <Spin />
        </div>
      ) : filtered.length === 0 ? (
        <Empty description={t('skins.catalog.empty')} />
      ) : (
        <div className="skins-catalog-grid">
          {filtered.map((entry) => (
            <article key={entry.id} className="skins-catalog-card">
              <img
                src={entry.preview_url}
                alt={entry.name}
                className="skins-catalog-avatar"
                loading="lazy"
                width={80}
                height={80}
              />
              <div className="skins-catalog-card-body">
                <Text strong className="skins-catalog-name">
                  {entry.name}
                </Text>
                <Text type="secondary" className="skins-catalog-source">
                  {t('skins.catalog.sourceMojang')}
                </Text>
                <Button
                  type="primary"
                  size="small"
                  loading={applyingId === entry.id}
                  onClick={() => void applyCatalog(entry)}
                >
                  {t('skins.catalog.select')}
                </Button>
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
