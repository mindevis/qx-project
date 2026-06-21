import { Segmented } from 'antd';
import { useI18n } from '@/i18n/I18nContext';
import type { Locale } from '@/i18n';

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n();

  return (
    <Segmented
      size="small"
      aria-label={t('language.label')}
      value={locale}
      onChange={(value) => setLocale(value as Locale)}
      options={[
        { label: 'RU', value: 'ru' },
        { label: 'EN', value: 'en' },
      ]}
    />
  );
}
