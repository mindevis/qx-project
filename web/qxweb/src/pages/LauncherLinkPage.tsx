import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Alert, Button, Card, Space, Typography } from 'antd';
import { CheckCircleOutlined, LinkOutlined } from '@ant-design/icons';
import { api, clearGuestSession, saveGuestSession, saveLinkedDevice } from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { logger } from '@/lib/logger';

export function LauncherLinkPage() {
  const { t } = useI18n();
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
      setError(t('launcherLink.missingDevice'));
    }
  }, [deviceId, t]);

  const handleLink = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.linkDevice({ device_id: deviceId });
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
      setError(e instanceof Error ? e.message : t('launcherLink.linkFailed'));
    } finally {
      setLoading(false);
    }
  };

  if (!deviceId) {
    return (
      <Alert
        type="error"
        message={t('launcherLink.linkError')}
        description={error ?? t('launcherLink.invalidLink')}
        action={
          <Link to="/launcher">
            <Button size="small">{t('launcherLink.toLauncher')}</Button>
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
            {t('launcherLink.deviceLinked')}
          </Typography.Title>
          <Typography.Paragraph>
            {ownerType === 'user' ? t('launcherLink.linkedAsUser') : t('launcherLink.linkedAsGuest')}
          </Typography.Paragraph>
          <Link to="/launcher">
            <Button type="primary">{t('launcherLink.goToInstances')}</Button>
          </Link>
        </Space>
      </Card>
    );
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 520 }}>
      <Typography.Title level={2}>{t('launcherLink.title')}</Typography.Title>
      <Typography.Paragraph type="secondary">{t('launcherLink.intro')}</Typography.Paragraph>

      {error && <Alert type="error" message={error} showIcon />}

      {isAuthenticated ? (
        <Card title={t('launcherLink.linkToAccount')}>
          <Space direction="vertical" style={{ width: '100%' }}>
            <Typography.Paragraph style={{ marginBottom: 0 }}>
              {t('launcherLink.linkToAccountHint')}
            </Typography.Paragraph>
            <Button
              type="primary"
              icon={<LinkOutlined />}
              loading={loading}
              onClick={() => handleLink()}
            >
              {t('launcherLink.linkDevice')}
            </Button>
          </Space>
        </Card>
      ) : (
        <>
          <Card title={t('launcherLink.guestMode')}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Typography.Paragraph style={{ marginBottom: 0 }}>
                {t('launcherLink.guestHint')}
              </Typography.Paragraph>
              <Button
                type="primary"
                loading={loading}
                icon={<LinkOutlined />}
                onClick={() => handleLink()}
              >
                {t('launcherLink.continueAsGuest')}
              </Button>
            </Space>
          </Card>
          <Card title={t('launcherLink.orSignIn')}>
            <Button onClick={() => openAuthModal('login')}>{t('auth.signIn')}</Button>
          </Card>
        </>
      )}
    </Space>
  );
}
