import { useI18n } from '@/i18n/I18nContext';
import type { Locale } from '@/i18n';
import { SegmentedControl } from './SegmentedControl';

const LOCALES: Locale[] = ['ru', 'en'];

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n();

  return (
    <SegmentedControl
      className="lang-switcher"
      value={locale}
      onChange={setLocale}
      groupLabel={t('language.label')}
      options={LOCALES.map((value) => ({
        value,
        label: value.toUpperCase(),
        ariaLabel: t(value === 'ru' ? 'language.ru' : 'language.en'),
      }))}
    />
  );
}
