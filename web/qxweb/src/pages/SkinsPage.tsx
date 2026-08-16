import { useState } from 'react';
import { Button, Result, Spin, Typography } from 'antd';
import { LoginOutlined, SkinOutlined } from '@ant-design/icons';
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
        <div className="skins-hero-ambient" aria-hidden>
          <span className="skins-hero-blob skins-hero-blob--1" />
          <span className="skins-hero-blob skins-hero-blob--2" />
          <span className="skins-hero-blob skins-hero-blob--3" />
          <span className="skins-hero-grid-pattern" />
        </div>

        <div className="skins-hero-inner">
          <div className="skins-hero-content">
            <p className="skins-hero-badge">{t('skins.badge')}</p>
            <Title level={1} className="skins-hero-title">
              {highlightMinecraft(t('skins.title'))}
            </Title>
            <Paragraph className="skins-hero-subtitle">{t('skins.subtitle')}</Paragraph>
          </div>

          <div className="skins-hero-visual" aria-hidden>
            <div className="skins-orbit">
              <div className="skins-orbit-ring skins-orbit-ring--outer" />
              <div className="skins-orbit-ring skins-orbit-ring--inner" />
              <div className="skins-orbit-core">
                <SkinOutlined />
              </div>
            </div>
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
