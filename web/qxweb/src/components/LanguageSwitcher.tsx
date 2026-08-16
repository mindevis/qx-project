import { Segmented } from 'antd';
import { useI18n } from '@/i18n/I18nContext';
import type { Locale } from '@/i18n';

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n();

  return (
    <Segmented<Locale>
      className="lang-switcher"
      size="small"
      value={locale}
      aria-label={t('language.label')}
      onChange={setLocale}
      options={[
        { value: 'ru', label: 'RU', title: t('language.ru') },
        { value: 'en', label: 'EN', title: t('language.en') },
      ]}
    />
  );
}
