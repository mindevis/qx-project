import { Link } from 'react-router-dom';
import { Button, Card, Space, Typography } from 'antd';
import {
  AppstoreOutlined,
  CloudServerOutlined,
  DesktopOutlined,
  LoginOutlined,
  RocketOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import './HomePage.css';

const { Title, Paragraph, Text } = Typography;

export function highlightMinecraft(title: string) {
  const marker = 'Minecraft';
  const index = title.indexOf(marker);
  if (index === -1) {
    return title;
  }

  return (
    <>
      {title.slice(0, index)}
      <span className="home-title-highlight">{marker}</span>
      {title.slice(index + marker.length)}
    </>
  );
}

export function HomePage() {
  const { t } = useI18n();
  const { isAuthenticated } = useAuth();

  const features = [
    {
      key: 'qxlauncher',
      icon: <DesktopOutlined />,
      title: t('home.qxlauncherTitle'),
      body: t('home.qxlauncherBody'),
      href: '/launcher',
    },
    {
      key: 'qxmods',
      icon: <AppstoreOutlined />,
      title: t('home.qxmodsTitle'),
      body: t('home.qxmodsBody'),
      href: '/launcher',
    },
    {
      key: 'qxagent',
      icon: <CloudServerOutlined />,
      title: t('home.qxagentTitle'),
      body: t('home.qxagentBody'),
      href: '/servers',
    },
  ] as const;

  const heroTags = [
    { key: 'launcher', label: t('home.heroTagLauncher') },
    { key: 'agent', label: t('home.heroTagAgent') },
  ];

  const benefits = [
    {
      key: 'account',
      icon: <UserOutlined />,
      title: t('home.benefitAccount'),
      desc: t('home.benefitAccountDesc'),
    },
    {
      key: 'dedicated',
      icon: <SafetyCertificateOutlined />,
      title: t('home.benefitDedicated'),
      desc: t('home.benefitDedicatedDesc'),
    },
    {
      key: 'vanilla',
      icon: <AppstoreOutlined />,
      title: t('home.benefitVanilla'),
      desc: t('home.benefitVanillaDesc'),
    },
  ];

  const steps = [
    { icon: <LoginOutlined />, title: t('home.step1Title'), body: t('home.step1Body') },
    { icon: <DesktopOutlined />, title: t('home.step2Title'), body: t('home.step2Body') },
    { icon: <RocketOutlined />, title: t('home.step3Title'), body: t('home.step3Body') },
  ];

  return (
    <div className="home-page">
      <noscript>
        <p className="home-noscript">{t('seo.noscriptSummary')}</p>
      </noscript>
      <section className="home-hero" aria-labelledby="home-hero-title">
        <div className="home-hero-ambient" aria-hidden>
          <span className="home-hero-blob home-hero-blob--1" />
          <span className="home-hero-blob home-hero-blob--2" />
          <span className="home-hero-blob home-hero-blob--3" />
          <span className="home-hero-grid-pattern" />
        </div>

        <div className="home-hero-grid">
          <div className="home-hero-content">
            <span className="home-badge">{t('home.badge')}</span>
            <Title level={1} id="home-hero-title" className="home-title">
              {highlightMinecraft(t('home.title'))}
            </Title>
            <Paragraph className="home-subtitle">{t('home.subtitle')}</Paragraph>
            <Paragraph type="secondary" className="home-intro">
              {t('home.intro')}
            </Paragraph>

            <div className="home-hero-tags" aria-label={t('home.badge')}>
              {heroTags.map((tag) => (
                <span key={tag.key} className={`home-hero-tag home-hero-tag--${tag.key}`}>
                  {tag.label}
                </span>
              ))}
            </div>

            <Space size="middle" wrap className="home-cta-row">
              <Link to="/launcher">
                <Button type="primary" size="large" icon={<RocketOutlined />}>
                  {t('home.ctaLauncher')}
                </Button>
              </Link>
              {isAuthenticated && (
                <Link to="/servers">
                  <Button size="large">{t('home.ctaServers')}</Button>
                </Link>
              )}
            </Space>
          </div>

          <div className="home-hero-visual" aria-hidden>
            <div className="home-orbit">
              <div className="home-orbit-ring home-orbit-ring--outer" />
              <div className="home-orbit-ring home-orbit-ring--inner" />
              <div className="home-orbit-core">
                <span className="home-orbit-logo">QX</span>
              </div>
              {features.map((feature, index) => (
                <div
                  key={feature.key}
                  className={`home-float-card home-float-card--${feature.key} home-float-card--pos-${index}`}
                >
                  <span className="home-float-card-icon">{feature.icon}</span>
                  <span className="home-float-card-title">{feature.title}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="home-section">
        <div className="home-benefits">
          {benefits.map((item) => (
            <div key={item.key} className={`home-benefit-card home-benefit-card--${item.key}`}>
              <div className="home-benefit-icon">{item.icon}</div>
              <Text strong className="home-benefit-title">
                {item.title}
              </Text>
              <Paragraph type="secondary" className="home-benefit-desc">
                {item.desc}
              </Paragraph>
            </div>
          ))}
        </div>
      </section>

      <section className="home-section home-section--features">
        <div className="home-section-header">
          <span className="home-section-eyebrow">{t('home.badge')}</span>
          <Title level={2} className="home-section-title">
            {t('home.featuresTitle')}
          </Title>
          <Paragraph type="secondary" className="home-section-lead">
            {t('home.featuresSubtitle')}
          </Paragraph>
        </div>
        <div className="home-features">
          {features.map((feature) => (
            <Card
              key={feature.key}
              className={`home-feature-card home-feature-card--${feature.key}`}
              variant="borderless"
            >
              <div className="home-feature-icon">{feature.icon}</div>
              <Title level={4} className="home-feature-title">
                {feature.title}
              </Title>
              <Paragraph type="secondary" className="home-feature-body">
                {feature.body}
              </Paragraph>
              <Link to={feature.href} className="home-feature-link">
                {t('common.open')} →
              </Link>
            </Card>
          ))}
        </div>
      </section>

      <section className="home-section home-section--steps">
        <div className="home-section-header">
          <Title level={2} className="home-section-title">
            {t('home.stepsTitle')}
          </Title>
        </div>
        <div className="home-steps">
          {steps.map((step) => (
            <div key={step.title} className="home-step">
              <span className="home-step-icon">{step.icon}</span>
              <Title level={5} className="home-step-title">
                {step.title}
              </Title>
              <Paragraph type="secondary" className="home-step-body">
                {step.body}
              </Paragraph>
            </div>
          ))}
        </div>
      </section>

      <section className="home-cta-band">
        <div className="home-cta-band-card">
          <div className="home-cta-band-inner">
            <Title level={2} className="home-cta-band-title">
              {t('home.ctaTitle')}
            </Title>
            <Paragraph type="secondary" className="home-cta-band-body">
              {t('home.ctaBody')}
            </Paragraph>
            <Space size="middle" wrap className="home-cta-band-actions">
              <Link to="/launcher">
                <Button type="primary" size="large" icon={<RocketOutlined />}>
                  {t('home.ctaLauncher')}
                </Button>
              </Link>
            </Space>
          </div>
        </div>
      </section>
    </div>
  );
}
