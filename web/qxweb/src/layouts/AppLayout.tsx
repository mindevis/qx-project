import { useEffect, useState, type CSSProperties } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { Layout, Menu, Button, Space, Typography, Spin, Tooltip, theme } from 'antd';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useBackendStatus } from '@/backend/BackendStatusContext';
import { useI18n } from '@/i18n/I18nContext';
import { useTheme } from '@/theme/ThemeContext';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import { ThemeToggle } from '@/components/ThemeToggle';
import { BackendUnavailableNotification } from '@/components/BackendUnavailableNotification';
import { RouteSeo } from '@/components/RouteSeo';
import { UserMenu } from '@/components/UserMenu';
import './AppLayout.css';

const { Header, Content, Footer } = Layout;

const SCROLL_THRESHOLD = 16;

export function AppLayout() {
  const { user, loading, isAuthenticated, logout } = useAuth();
  const { openAuthModal } = useAuthModal();
  const { available: backendAvailable } = useBackendStatus();
  const { t } = useI18n();
  const { isDark } = useTheme();
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const location = useLocation();
  const isLandingPage =
    location.pathname === '/' ||
    location.pathname === '/launcher' ||
    location.pathname === '/launcher/link' ||
    location.pathname.startsWith('/launcher/instances/') ||
    location.pathname === '/monitoring' ||
    location.pathname === '/servers' ||
    location.pathname.startsWith('/servers/');
  const footerClassName =
    location.pathname === '/'
      ? 'app-footer app-footer--landing app-footer--landing-home'
      : location.pathname === '/launcher' ||
          location.pathname === '/launcher/link' ||
          location.pathname.startsWith('/launcher/instances/')
        ? 'app-footer app-footer--landing app-footer--landing-launcher'
        : location.pathname === '/monitoring'
          ? 'app-footer app-footer--landing app-footer--landing-monitoring'
          : location.pathname === '/servers' || location.pathname.startsWith('/servers/')
            ? 'app-footer app-footer--landing app-footer--landing-servers'
            : 'app-footer';
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => {
      setScrolled(window.scrollY > SCROLL_THRESHOLD);
    };

    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, [location.pathname]);

  const headerClassName = [
    'app-header',
    'app-header--sticky',
    isLandingPage && 'app-header--landing',
    scrolled && 'app-header--scrolled',
  ]
    .filter(Boolean)
    .join(' ');

  const menuItems = [
    { key: '/', label: <Link to="/">{t('layout.navHome')}</Link> },
    { key: '/launcher', label: <Link to="/launcher">{t('layout.navLauncher')}</Link> },
    { key: '/monitoring', label: <Link to="/monitoring">{t('layout.navMonitoring')}</Link> },
    ...(isAuthenticated
      ? [
          { key: '/skins', label: <Link to="/skins">{t('layout.navSkins')}</Link> },
          { key: '/servers', label: <Link to="/servers">{t('layout.navServers')}</Link> },
        ]
      : []),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <RouteSeo />
      <BackendUnavailableNotification />
      <Header
        className={headerClassName}
        style={
          {
            '--app-header-bg': token.colorBgContainer,
            '--app-header-border': token.colorBorderSecondary,
          } as CSSProperties
        }
      >
        <Link to="/" className="app-header-brand">
          <Typography.Title level={4} style={{ color: token.colorText, margin: 0 }}>
            QXSystem
          </Typography.Title>
        </Link>
        <Menu
          theme={isDark ? 'dark' : 'light'}
          mode="horizontal"
          selectable={false}
          items={menuItems}
          style={{ flex: 1, minWidth: 0, background: 'transparent' }}
        />
        <Space className="app-header-controls" size={8}>
          <ThemeToggle />
          <LanguageSwitcher />
          {loading ? (
            <Spin size="small" />
          ) : isAuthenticated && user ? (
            <UserMenu
              user={user}
              onLogout={async () => {
                await logout();
                navigate('/');
              }}
            />
          ) : (
            <Tooltip title={!backendAvailable ? t('auth.backendUnavailable') : undefined}>
              <span>
                <Button
                  type="primary"
                  className="app-header-login-btn"
                  disabled={!backendAvailable}
                  onClick={() => openAuthModal('login')}
                >
                  {t('common.login')}
                </Button>
              </span>
            </Tooltip>
          )}
        </Space>
      </Header>
      <Content
        className={isLandingPage ? 'app-content--landing' : 'app-content--main'}
        style={{ padding: isLandingPage ? '0' : '24px 48px' }}
      >
        <main>
          <Outlet />
        </main>
      </Content>
      <Footer className={footerClassName}>
        {t('layout.footer')}
      </Footer>
    </Layout>
  );
}
