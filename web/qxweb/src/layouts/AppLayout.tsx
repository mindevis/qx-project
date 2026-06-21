import { useEffect, useState, type CSSProperties } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { Layout, Menu, Button, Space, Typography, Spin, theme } from 'antd';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { useTheme } from '@/theme/ThemeContext';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import { ThemeToggle } from '@/components/ThemeToggle';
import { BackendUnavailableNotification } from '@/components/BackendUnavailableNotification';
import { UserMenu } from '@/components/UserMenu';
import './AppLayout.css';

const { Header, Content, Footer } = Layout;

const SCROLL_THRESHOLD = 16;

export function AppLayout() {
  const { user, loading, isAuthenticated, logout } = useAuth();
  const { openAuthModal } = useAuthModal();
  const { t } = useI18n();
  const { isDark } = useTheme();
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const location = useLocation();
  const isHome = location.pathname === '/';
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
    isHome && 'app-header--home',
    scrolled && 'app-header--scrolled',
  ]
    .filter(Boolean)
    .join(' ');

  const menuItems = [
    { key: '/', label: <Link to="/">{t('layout.navHome')}</Link> },
    { key: '/launcher', label: <Link to="/launcher">{t('layout.navLauncher')}</Link> },
    ...(isAuthenticated
      ? [{ key: '/servers', label: <Link to="/servers">{t('layout.navServers')}</Link> }]
      : []),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
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
        <Typography.Title level={4} style={{ color: token.colorText, margin: 0 }}>
          QXProject
        </Typography.Title>
        <Menu
          theme={isDark ? 'dark' : 'light'}
          mode="horizontal"
          selectable={false}
          items={menuItems}
          style={{ flex: 1, minWidth: 0, background: 'transparent' }}
        />
        <Space>
          <LanguageSwitcher />
          <ThemeToggle />
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
            <Button type="primary" onClick={() => openAuthModal('login')}>
              {t('common.login')}
            </Button>
          )}
        </Space>
      </Header>
      <Content
        className={isHome ? 'app-content--home' : undefined}
        style={{ padding: isHome ? '0 48px 24px' : '24px 48px' }}
      >
        <Outlet />
      </Content>
      <Footer style={{ textAlign: 'center' }}>{t('layout.footer')}</Footer>
    </Layout>
  );
}
