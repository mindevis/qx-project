import { Navigate } from 'react-router-dom';
import { Button, Card, Descriptions, Space, Spin, Tooltip } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useAuth } from '@/auth/AuthContext';
import { ChangeEmailModal } from '@/components/ChangeEmailModal';
import { ChangePasswordModal } from '@/components/ChangePasswordModal';
import { useMessage } from '@/hooks/useMessage';
import { useI18n } from '@/i18n/I18nContext';

export function ProfilePage() {
  const { user, loading, isAuthenticated, refreshProfile } = useAuth();
  const message = useMessage();
  const { t } = useI18n();
  const [emailModalOpen, setEmailModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);

  if (loading) {
    return <Spin size="large" />;
  }
  if (!isAuthenticated || !user) {
    return <Navigate to="/" replace />;
  }

  return (
    <>
      <Card title={t('profile.title')} style={{ maxWidth: 560 }}>
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
