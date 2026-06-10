import { Link, Outlet, useNavigate } from 'react-router-dom';
import { Layout, Menu, Button, Space, Typography, Spin, theme } from 'antd';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useTheme } from '@/theme/ThemeContext';
import { ThemeToggle } from '@/components/ThemeToggle';
import { UserMenu } from '@/components/UserMenu';

const { Header, Content, Footer } = Layout;

export function AppLayout() {
  const { user, loading, isAuthenticated, logout } = useAuth();
  const { openAuthModal } = useAuthModal();
  const { isDark } = useTheme();
  const { token } = theme.useToken();
  const navigate = useNavigate();

  const menuItems = [
    { key: '/', label: <Link to="/">Главная</Link> },
    { key: '/launcher', label: <Link to="/launcher">Лаунчер</Link> },
    ...(isAuthenticated ? [{ key: '/servers', label: <Link to="/servers">Серверы</Link> }] : []),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 24,
          background: token.colorBgContainer,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          paddingInline: 24,
        }}
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
              Вход
            </Button>
          )}
        </Space>
      </Header>
      <Content style={{ padding: '24px 48px' }}>
        <Outlet />
      </Content>
      <Footer style={{ textAlign: 'center' }}>
        QXProject — Minecraft ecosystem (MVP Phase 0)
      </Footer>
    </Layout>
  );
}
