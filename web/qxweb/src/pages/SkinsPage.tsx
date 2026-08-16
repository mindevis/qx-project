import { useState } from 'react';
import { Button, Result, Spin, Typography } from 'antd';
import { LoginOutlined } from '@ant-design/icons';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { CosmeticsPanel } from '@/components/CosmeticsPanel';
import { SkinCatalogPanel } from '@/components/SkinCatalogPanel';
import { useI18n } from '@/i18n/I18nContext';
import { highlightMinecraft } from '@/pages/HomePage';
import './SkinsPage.css';

const { Title, Paragraph } = Typography;

export function SkinsPage() {
  const { loading, isAuthenticated } = useAuth();
  const { openAuthModal } = useAuthModal();
  const { t } = useI18n();
  const [panelKey, setPanelKey] = useState(0);

  if (loading) {
    return <Spin size="large" />;
  }
  if (!isAuthenticated) {
    return (
      <Result
        status="403"
        title={t('skins.authRequiredTitle')}
        subTitle={t('skins.authRequiredDesc')}
        extra={
          <Button type="primary" icon={<LoginOutlined />} onClick={() => openAuthModal('login')}>
            {t('auth.signIn')}
          </Button>
        }
      />
    );
  }

  return (
    <div className="skins-page">
      <section className="skins-hero">
        <div className="skins-hero-inner">
          <div className="skins-hero-content">
            <Title level={1} className="skins-hero-title">
              {highlightMinecraft(t('skins.title'))}
            </Title>
            <Paragraph type="secondary" className="skins-hero-subtitle">{t('skins.subtitle')}</Paragraph>
          </div>
        </div>
      </section>

      <div className="skins-body">
        <section className="skins-section skins-section--catalog">
          <Title level={3} className="skins-section-title">
            {t('skins.catalog.sectionTitle')}
          </Title>
          <Paragraph type="secondary" className="skins-section-lead">
            {t('skins.catalog.sectionLead')}
          </Paragraph>
          <SkinCatalogPanel onApplied={() => setPanelKey((k) => k + 1)} />
        </section>

        <hr className="skins-section-divider" aria-hidden />

        <section className="skins-section skins-section--equip">
          <Title level={3} className="skins-section-title">
            {t('skins.equip.sectionTitle')}
          </Title>
          <div className="skins-equip-card">
            <CosmeticsPanel key={panelKey} embedded />
          </div>
        </section>
      </div>
    </div>
  );
}
