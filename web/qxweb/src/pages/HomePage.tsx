import { Link } from 'react-router-dom';
import { Button, Card, Space, Typography } from 'antd';
import {
  CloudServerOutlined,
  DesktopOutlined,
  GlobalOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import { LauncherDownloadButton } from '@/components/LauncherDownloadButton';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import './HomePage.css';

const { Title, Paragraph, Text } = Typography;

export function HomePage() {
  const { t } = useI18n();
  const { isAuthenticated } = useAuth();

  const features = [
    {
      key: 'qxweb',
      icon: <GlobalOutlined />,
      title: t('home.qxwebTitle'),
      body: t('home.qxwebBody'),
    },
    {
      key: 'qxlauncher',
      icon: <DesktopOutlined />,
      title: t('home.qxlauncherTitle'),
      body: t('home.qxlauncherBody'),
    },
    {
      key: 'qxagent',
      icon: <CloudServerOutlined />,
      title: t('home.qxagentTitle'),
      body: t('home.qxagentBody'),
    },
  ];

  const benefits = [
    { title: t('home.benefitAccount'), desc: t('home.benefitAccountDesc') },
    { title: t('home.benefitByos'), desc: t('home.benefitByosDesc') },
    { title: t('home.benefitVanilla'), desc: t('home.benefitVanillaDesc') },
  ];

  const steps = [
    { title: t('home.step1Title'), body: t('home.step1Body') },
    { title: t('home.step2Title'), body: t('home.step2Body') },
    { title: t('home.step3Title'), body: t('home.step3Body') },
  ];

  return (
    <div className="home-page">
      <section className="home-hero">
        <div className="home-hero-inner">
          <span className="home-badge">{t('home.badge')}</span>
          <Title level={1} className="home-title">
            {t('home.title')}
          </Title>
          <Paragraph className="home-subtitle">{t('home.subtitle')}</Paragraph>
          <Paragraph type="secondary" className="home-intro">
            {t('home.intro')}
          </Paragraph>
          <Space size="middle" wrap className="home-cta-row">
            <Link to="/launcher">
              <Button type="primary" size="large" icon={<RocketOutlined />}>
                {t('home.ctaLauncher')}
              </Button>
            </Link>
            <LauncherDownloadButton type="default" />
            {isAuthenticated && (
              <Link to="/servers">
                <Button size="large">{t('home.ctaServers')}</Button>
              </Link>
            )}
          </Space>
        </div>
      </section>

      <section className="home-section">
        <div className="home-benefits">
          {benefits.map((item) => (
            <div key={item.title} className="home-benefit-card">
              <Text strong className="home-benefit-title">
                {item.title}
              </Text>
              <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                {item.desc}
              </Paragraph>
            </div>
          ))}
        </div>
      </section>

      <section className="home-section">
        <div className="home-section-header">
          <Title level={2} className="home-section-title">
            {t('home.featuresTitle')}
          </Title>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t('home.featuresSubtitle')}
          </Paragraph>
        </div>
        <div className="home-features">
          {features.map((feature) => (
            <Card key={feature.key} className="home-feature-card" bordered={false}>
              <div className="home-feature-icon">{feature.icon}</div>
              <Title level={4} style={{ marginTop: 0 }}>
                {feature.title}
              </Title>
              <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                {feature.body}
              </Paragraph>
            </Card>
          ))}
        </div>
      </section>

      <section className="home-section">
        <div className="home-section-header">
          <Title level={2} className="home-section-title">
            {t('home.stepsTitle')}
          </Title>
        </div>
        <div className="home-steps">
          {steps.map((step, index) => (
            <div key={step.title} className="home-step">
              <span className="home-step-num">{index + 1}</span>
              <Title level={5} style={{ marginTop: 0 }}>
                {step.title}
              </Title>
              <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                {step.body}
              </Paragraph>
            </div>
          ))}
        </div>
      </section>

      <section className="home-cta-band">
        <div className="home-cta-band-inner">
          <Title level={2} className="home-cta-band-title">
            {t('home.ctaTitle')}
          </Title>
          <Paragraph type="secondary" className="home-cta-band-body">
            {t('home.ctaBody')}
          </Paragraph>
          <Space size="middle" wrap style={{ justifyContent: 'center' }}>
            <Link to="/launcher">
              <Button type="primary" size="large" icon={<RocketOutlined />}>
                {t('home.ctaLauncher')}
              </Button>
            </Link>
            <LauncherDownloadButton />
          </Space>
        </div>
      </section>
    </div>
  );
}
