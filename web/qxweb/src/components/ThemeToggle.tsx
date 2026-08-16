import { Segmented } from 'antd';
import { MoonOutlined, SunOutlined } from '@ant-design/icons';
import { useI18n } from '@/i18n/I18nContext';
import { useTheme, type ThemeMode } from '@/theme/ThemeContext';

export function ThemeToggle() {
  const { mode, setMode } = useTheme();
  const { t } = useI18n();

  return (
    <Segmented<ThemeMode>
      className="theme-switcher"
      size="small"
      value={mode}
      aria-label={t('theme.label')}
      onChange={setMode}
      options={[
        {
          value: 'light',
          label: (
            <span aria-label={t('theme.light')}>
              <SunOutlined />
            </span>
          ),
        },
        {
          value: 'dark',
          label: (
            <span aria-label={t('theme.dark')}>
              <MoonOutlined />
            </span>
          ),
        },
      ]}
    />
  );
}
