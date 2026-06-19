import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Alert, Button, Card, Form, Input, Space, Typography } from 'antd';
import { CheckCircleOutlined, LinkOutlined } from '@ant-design/icons';
import { api, clearGuestSession, saveGuestSession, saveLinkedDevice } from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { logger } from '@/lib/logger';

export function LauncherLinkPage() {
  const [params] = useSearchParams();
  const deviceId = params.get('device')?.trim() ?? '';
  const { isAuthenticated } = useAuth();
  const { openAuthModal } = useAuthModal();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [linked, setLinked] = useState(false);
  const [ownerType, setOwnerType] = useState<string | null>(null);

  useEffect(() => {
    if (!deviceId) {
      setError('Не указан идентификатор устройства (параметр device).');
    }
  }, [deviceId]);

  const handleLink = async (userCode?: string) => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.linkDevice({
        device_id: deviceId,
        user_code: userCode || undefined,
      });
      setOwnerType(result.owner_type);
      setLinked(true);
      saveLinkedDevice(deviceId);
      if (result.owner_type === 'user') {
        clearGuestSession();
      } else if (result.guest_token) {
        saveGuestSession({
          guest_token: result.guest_token,
          expires_in: result.guest_expires_in ?? 86400,
        });
      }
      logger.info('device linked', { deviceId, ownerType: result.owner_type });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось связать устройство');
    } finally {
      setLoading(false);
    }
  };

  if (!deviceId) {
    return (
      <Alert
        type="error"
        message="Ошибка связывания"
        description={error ?? 'Некорректная ссылка.'}
        action={
          <Link to="/launcher">
            <Button size="small">К лаунчеру</Button>
          </Link>
        }
      />
    );
  }

  if (linked) {
    return (
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Typography.Title level={3}>
            <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
            Устройство связано
          </Typography.Title>
          <Typography.Paragraph>
            {ownerType === 'user'
              ? 'QXLauncher привязан к вашему аккаунту. Можно создавать инстансы на сайте.'
              : 'Гостевая сессия создана. Можно создавать Vanilla-инстансы на сайте.'}
          </Typography.Paragraph>
          <Link to="/launcher">
            <Button type="primary">Перейти к инстансам</Button>
          </Link>
        </Space>
      </Card>
    );
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 520 }}>
      <Typography.Title level={2}>Связать QXLauncher</Typography.Title>
      <Typography.Paragraph type="secondary">
        Подтвердите привязку устройства <Typography.Text code>{deviceId}</Typography.Text> к
        аккаунту на сайте.
      </Typography.Paragraph>

      {error && <Alert type="error" message={error} showIcon />}

      {isAuthenticated ? (
        <Card title="Привязать к аккаунту">
          <Space direction="vertical" style={{ width: '100%' }}>
            <Typography.Paragraph style={{ marginBottom: 0 }}>
              Вы вошли в аккаунт. Нажмите кнопку, чтобы связать устройство.
            </Typography.Paragraph>
            <Button
              type="primary"
              icon={<LinkOutlined />}
              loading={loading}
              onClick={() => handleLink()}
            >
              Связать устройство
            </Button>
          </Space>
        </Card>
      ) : (
        <>
          <Card title="Гостевой режим">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Typography.Paragraph style={{ marginBottom: 0 }}>
                Без регистрации — только Vanilla-инстансы.
              </Typography.Paragraph>
              <Form layout="vertical" onFinish={(v) => handleLink(v.user_code)}>
                <Form.Item name="user_code" label="Код с экрана лаунчера (необязательно)">
                  <Input placeholder="ABCD-1234" autoComplete="off" />
                </Form.Item>
                <Button type="primary" htmlType="submit" loading={loading} icon={<LinkOutlined />}>
                  Продолжить как гость
                </Button>
              </Form>
            </Space>
          </Card>
          <Card title="Или войдите в аккаунт">
            <Button onClick={() => openAuthModal('login')}>Войти</Button>
          </Card>
        </>
      )}
    </Space>
  );
}
