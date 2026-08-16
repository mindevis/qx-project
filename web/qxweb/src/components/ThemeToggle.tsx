import { MoonOutlined, SunOutlined } from '@ant-design/icons';
import { useI18n } from '@/i18n/I18nContext';
import { useTheme, type ThemeMode } from '@/theme/ThemeContext';
import { SegmentedControl } from './SegmentedControl';

const MODES: ThemeMode[] = ['light', 'dark'];

export function ThemeToggle() {
  const { mode, setMode } = useTheme();
  const { t } = useI18n();

  return (
    <SegmentedControl
      className="theme-switcher"
      iconOnly
      value={mode}
      onChange={setMode}
      groupLabel={t('theme.label')}
      options={MODES.map((value) => ({
        value,
        label: value === 'light' ? <SunOutlined /> : <MoonOutlined />,
        ariaLabel: t(value === 'light' ? 'theme.light' : 'theme.dark'),
      }))}
    />
  );
}
