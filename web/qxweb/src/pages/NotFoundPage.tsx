import { Link } from 'react-router-dom';
import { Button, Result } from 'antd';
import { useI18n } from '@/i18n/I18nContext';

export function NotFoundPage() {
  const { t } = useI18n();

  return (
    <Result
      status="404"
      title={t('common.pageNotFound')}
      subTitle={t('common.pageNotFoundDesc')}
      extra={
        <Link to="/">
          <Button type="primary">{t('common.backHome')}</Button>
        </Link>
      }
    />
  );
}
