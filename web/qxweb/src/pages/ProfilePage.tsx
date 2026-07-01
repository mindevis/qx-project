import { Link, Navigate, useSearchParams } from 'react-router-dom';
import { Button, Card, Descriptions, Popconfirm, Space, Spin, Tooltip, Typography } from 'antd';
import { EditOutlined, LinkOutlined, SkinOutlined } from '@ant-design/icons';
import { useCallback, useEffect, useState } from 'react';
import { useAuth } from '@/auth/AuthContext';
import { ChangeEmailModal } from '@/components/ChangeEmailModal';
import { ChangePasswordModal } from '@/components/ChangePasswordModal';
import { useMessage } from '@/hooks/useMessage';
import { useI18n } from '@/i18n/I18nContext';
import { api, type MojangLinkStatus } from '@/api/client';
import { logger } from '@/lib/logger';

export function ProfilePage() {
  const { user, loading, isAuthenticated, refreshProfile } = useAuth();
  const message = useMessage();
  const { t } = useI18n();
  const [searchParams, setSearchParams] = useSearchParams();
  const [emailModalOpen, setEmailModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [mojangStatus, setMojangStatus] = useState<MojangLinkStatus | null>(null);
  const [mojangLoading, setMojangLoading] = useState(false);
  const [mojangActionLoading, setMojangActionLoading] = useState(false);

  const loadMojangStatus = useCallback(async () => {
    setMojangLoading(true);
    try {
      const status = await api.mojangStatus();
      setMojangStatus(status);
    } catch (e) {
      logger.warn('failed to load mojang status', { error: String(e) });
      setMojangStatus(null);
    } finally {
      setMojangLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isAuthenticated) {
      void loadMojangStatus();
    }
  }, [isAuthenticated, loadMojangStatus]);

  useEffect(() => {
    if (searchParams.get('mojang') === 'linked') {
      message.success(t('profile.mojangLinkSuccess'));
      const next = new URLSearchParams(searchParams);
      next.delete('mojang');
      setSearchParams(next, { replace: true });
      void loadMojangStatus();
    }
  }, [loadMojangStatus, message, searchParams, setSearchParams, t]);

  const handleLinkMojang = async () => {
    setMojangActionLoading(true);
    try {
      const res = await api.startMojangOAuth();
      window.location.assign(res.authorization_url);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('profile.mojangLinkFailed'));
    } finally {
      setMojangActionLoading(false);
    }
  };

  const handleUnlinkMojang = async () => {
    setMojangActionLoading(true);
    try {
      await api.unlinkMojang();
      setMojangStatus({ linked: false });
      message.success(t('profile.mojangUnlinkSuccess'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('profile.mojangUnlinkFailed'));
    } finally {
      setMojangActionLoading(false);
    }
  };

  if (loading) {
    return <Spin size="large" />;
  }
  if (!isAuthenticated || !user) {
    return <Navigate to="/" replace />;
  }

  return (
    <>
      <Card title={t('profile.title')} style={{ maxWidth: 560, marginBottom: 16 }}>
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label={t('common.id')}>{user.id}</Descriptions.Item>
          <Descriptions.Item label={t('common.email')}>
            <Space>
              <span>{user.email}</span>
              <Tooltip title={t('profile.changeEmail')}>
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  aria-label={t('profile.changeEmail')}
                  onClick={() => setEmailModalOpen(true)}
                />
              </Tooltip>
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label={t('common.password')}>
            <Space>
              <span>••••••••</span>
              <Button size="small" onClick={() => setPasswordModalOpen(true)}>
                {t('profile.changePassword')}
              </Button>
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label={t('common.tier')}>{user.tier}</Descriptions.Item>
          <Descriptions.Item label={t('common.created')}>{user.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title={t('profile.mojangTitle')} style={{ maxWidth: 560 }}>
        {mojangLoading ? (
          <Spin />
        ) : mojangStatus?.linked ? (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label={t('profile.mojangUsername')}>
              {mojangStatus.username}
            </Descriptions.Item>
            <Descriptions.Item label={t('profile.mojangUuid')}>
              {mojangStatus.minecraft_uuid}
            </Descriptions.Item>
            {mojangStatus.linked_at ? (
              <Descriptions.Item label={t('profile.mojangLinkedAt')}>
                {mojangStatus.linked_at}
              </Descriptions.Item>
            ) : null}
            <Descriptions.Item label={t('common.done')}>
              <Popconfirm
                title={t('profile.mojangUnlinkConfirm')}
                onConfirm={() => void handleUnlinkMojang()}
              >
                <Button danger loading={mojangActionLoading}>
                  {t('profile.mojangUnlink')}
                </Button>
              </Popconfirm>
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <span>{t('profile.mojangHint')}</span>
            <Button
              type="primary"
              icon={<LinkOutlined />}
              loading={mojangActionLoading}
              onClick={() => void handleLinkMojang()}
            >
              {t('profile.mojangLink')}
            </Button>
          </Space>
        )}
      </Card>

      <Card title={t('profile.skinsTitle')} style={{ maxWidth: 560 }}>
        <Space orientation="vertical" size="middle">
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            {t('profile.skinsHint')}
          </Typography.Paragraph>
          <Link to="/skins">
            <Button type="primary" icon={<SkinOutlined />}>
              {t('profile.goToSkins')}
            </Button>
          </Link>
        </Space>
      </Card>

      <ChangeEmailModal
        open={emailModalOpen}
        currentEmail={user.email}
        onClose={() => setEmailModalOpen(false)}
        onSuccess={() => {
          void refreshProfile();
          message.success(t('profile.emailChanged'));
        }}
      />
      <ChangePasswordModal
        open={passwordModalOpen}
        onClose={() => setPasswordModalOpen(false)}
        onSuccess={() => message.success(t('profile.passwordChanged'))}
      />
    </>
  );
}
