import { Typography } from 'antd';
import { useI18n } from '@/i18n/I18nContext';

const { Text } = Typography;

const PRIVACY_URL = 'https://docs.qx-dev.ru/privacy/';

export function LauncherCodeSigningNotice() {
  const { t } = useI18n();

  return (
    <Text type="secondary" className="launcher-code-signing-notice">
      {t('launcher.codeSigningNotice')}{' '}
      <a href={PRIVACY_URL} target="_blank" rel="noreferrer">
        {t('launcher.privacyPolicyLink')}
      </a>
    </Text>
  );
}
