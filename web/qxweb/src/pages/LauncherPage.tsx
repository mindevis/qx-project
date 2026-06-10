import { Button, Card, Space, Typography } from 'antd';
import { DownloadOutlined, RocketOutlined } from '@ant-design/icons';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';

export function LauncherPage() {
  const { isAuthenticated } = useAuth();
  const { openAuthModal } = useAuthModal();

  return (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 720 }}>
      <Typography.Title level={2}>Лаунчер</Typography.Title>
      <Typography.Paragraph type="secondary">
        Запускайте Minecraft через браузер или установите QXLauncher на компьютер для работы из
        системного трея.
      </Typography.Paragraph>

      <Card title="QXLauncher для ПК">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            Десктопное приложение с автозапуском, офлайн-кэшем и нативной интеграцией. Скачивание
            будет доступно в Phase 1.
          </Typography.Paragraph>
          <Button icon={<DownloadOutlined />} disabled>
            Скачать QXLauncher (Phase 1)
          </Button>
        </Space>
      </Card>

      <Card title="Веб-лаунчер">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            Запуск игры прямо из браузера — без установки дополнительного ПО.
          </Typography.Paragraph>
          {isAuthenticated ? (
            <Button type="primary" icon={<RocketOutlined />} disabled>
              Открыть лаунчер (Phase 1)
            </Button>
          ) : (
            <Button type="primary" onClick={() => openAuthModal('login')}>
              Войти, чтобы открыть лаунчер
            </Button>
          )}
        </Space>
      </Card>
    </Space>
  );
}
