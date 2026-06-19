import { Card, Space, Typography } from 'antd';
import { LauncherDownloadButton } from '@/components/LauncherDownloadButton';

export function HomePage() {
  return (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 720 }}>
      <Typography.Title level={2}>Единая экосистема для Minecraft</Typography.Title>
      <Typography.Paragraph type="secondary">
        QXProject объединяет веб-панель, десктопный лаунчер и агент для управления сервером — один
        аккаунт, общие модпаки и настройки.
      </Typography.Paragraph>

      <LauncherDownloadButton type="primary" />

      <Card title="QXWeb">
        <Typography.Paragraph style={{ marginBottom: 0 }}>
          Панель и лаунчер в браузере: профиль, каталог серверов и запуск игры без установки
          приложения.
        </Typography.Paragraph>
      </Card>

      <Card title="QXLauncher">
        <Typography.Paragraph style={{ marginBottom: 0 }}>
          Настольное приложение в системном трее: автозапуск, офлайн-кэш и нативная интеграция с
          Windows, macOS и Linux.
        </Typography.Paragraph>
      </Card>

      <Card title="QXAgent">
        <Typography.Paragraph style={{ marginBottom: 0 }}>
          Агент на вашем сервере (BYOS): установка контента, мониторинг и удалённое управление
          инстансом Minecraft.
        </Typography.Paragraph>
      </Card>
    </Space>
  );
}
