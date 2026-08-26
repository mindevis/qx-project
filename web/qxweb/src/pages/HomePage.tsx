import { Link } from 'react-router-dom';
import { Button, Card, Space, Typography } from 'antd';
import type { ReactNode } from 'react';
import {
  AppstoreOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  DesktopOutlined,
  LoginOutlined,
  RobotOutlined,
  RocketOutlined,
  SafetyCertificateOutlined,
  SkinOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { DiscordInviteLink } from '@/components/DiscordInviteLink';
import './HomePage.css';

const { Title, Paragraph, Text } = Typography;

function isAccountOnlyPath(to: string) {
  return to === '/servers' || to.startsWith('/servers/') || to === '/skins';
}

function HomeSectionLink({
  to,
  className,
  children,
}: {
  to: string;
  className?: string;
  children: ReactNode;
}) {
  const { isAuthenticated } = useAuth();
  const { openAuthModal } = useAuthModal();
  if (!isAuthenticated && isAccountOnlyPath(to)) {
    return (
      <a
        href={to}
        className={className}
        onClick={(event) => {
          event.preventDefault();
          openAuthModal('login');
        }}
      >
        {children}
      </a>
    );
  }
  return (
    <Link to={to} className={className}>
      {children}
    </Link>
  );
}

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
    {
      key: 'qxskins',
      icon: <SkinOutlined />,
      title: t('home.qxskinsTitle'),
      body: t('home.qxskinsBody'),
      href: '/skins',
    },
  ] as const;

  const heroTags: Array<{ key: string; label: string; fresh?: boolean }> = [
    { key: 'launcher', label: t('home.heroTagLauncher') },
    { key: 'agent', label: t('home.heroTagAgent') },
    { key: 'skins', label: t('home.heroTagSkins') },
    { key: 'mysql', label: t('home.heroTagMysql'), fresh: true },
    { key: 'ollama', label: t('home.heroTagOllama'), fresh: true },
  ];

  const stackCards = [
    {
      key: 'mysql',
      icon: <DatabaseOutlined />,
      title: t('home.stackMysqlTitle'),
      body: t('home.stackMysqlBody'),
      pills: [t('home.stackMysqlPill1'), t('home.stackMysqlPill2'), t('home.stackMysqlPill3')],
    },
    {
      key: 'ollama',
      icon: <RobotOutlined />,
      title: t('home.stackOllamaTitle'),
      body: t('home.stackOllamaBody'),
      pills: [t('home.stackOllamaPill1'), t('home.stackOllamaPill2'), t('home.stackOllamaPill3')],
    },
  ] as const;

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
                <span
                  key={tag.key}
                  className={`home-hero-tag home-hero-tag--${tag.key}${tag.fresh ? ' home-hero-tag--fresh' : ''}`}
                >
                  {tag.label}
                  {tag.fresh ? (
                    <span className="home-hero-tag-new">{t('layout.navNewBadge')}</span>
                  ) : null}
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

      <section className="home-section home-section--stack" aria-labelledby="home-stack-title">
        <div className="home-section-header">
          <span className="home-section-eyebrow">{t('layout.navNewBadge')}</span>
          <Title level={2} id="home-stack-title" className="home-section-title">
            {t('home.stackTitle')}
          </Title>
          <Paragraph type="secondary" className="home-section-lead">
            {t('home.stackLead')}
          </Paragraph>
        </div>
        <div className="home-stack">
          {stackCards.map((card) => (
            <HomeSectionLink
              key={card.key}
              to="/servers"
              className={`home-stack-card home-stack-card--${card.key}`}
            >
              <span className="home-stack-badge">{t('layout.navNewBadge')}</span>
              <span className="home-stack-icon">{card.icon}</span>
              <Title level={3} className="home-stack-title">
                {card.title}
              </Title>
              <Paragraph className="home-stack-body">{card.body}</Paragraph>
              <div className="home-stack-pills">
                {card.pills.map((pill) => (
                  <span key={pill} className="home-stack-pill">
                    {pill}
                  </span>
                ))}
              </div>
              <span className="home-stack-link">
                {t('home.stackOpen')} →
              </span>
            </HomeSectionLink>
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
              <Title level={3} className="home-feature-title">
                {feature.title}
              </Title>
              <Paragraph type="secondary" className="home-feature-body">
                {feature.body}
              </Paragraph>
              <HomeSectionLink to={feature.href} className="home-feature-link">
                {t('common.open')} →
              </HomeSectionLink>
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
              <Title level={3} className="home-step-title">
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
              <DiscordInviteLink />
            </Space>
          </div>
        </div>
      </section>
    </div>
  );
}
