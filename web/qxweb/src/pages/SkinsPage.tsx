import { Navigate } from 'react-router-dom';
import { Spin, Typography } from 'antd';
import { useAuth } from '@/auth/AuthContext';
import { CosmeticsPanel } from '@/components/CosmeticsPanel';
import { useI18n } from '@/i18n/I18nContext';

const { Title, Paragraph } = Typography;

export function SkinsPage() {
  const { loading, isAuthenticated } = useAuth();
  const { t } = useI18n();

  if (loading) {
    return <Spin size="large" />;
  }
  if (!isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="skins-page">
      <Title level={2} style={{ marginTop: 0 }}>
        {t('skins.title')}
      </Title>
      <Paragraph type="secondary">{t('skins.subtitle')}</Paragraph>
      <CosmeticsPanel embedded />
    </div>
  );
}
