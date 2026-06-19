import { Navigate } from 'react-router-dom';
import { Button, Card, Descriptions, Space, Spin, Tooltip, message } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useAuth } from '@/auth/AuthContext';
import { ChangeEmailModal } from '@/components/ChangeEmailModal';
import { ChangePasswordModal } from '@/components/ChangePasswordModal';

export function ProfilePage() {
  const { user, loading, isAuthenticated, refreshProfile } = useAuth();
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
      <Card title="Профиль" style={{ maxWidth: 560 }}>
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="ID">{user.id}</Descriptions.Item>
          <Descriptions.Item label="Email">
            <Space>
              <span>{user.email}</span>
              <Tooltip title="Сменить email">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  aria-label="Сменить email"
                  onClick={() => setEmailModalOpen(true)}
                />
              </Tooltip>
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Пароль">
            <Space>
              <span>••••••••</span>
              <Button size="small" onClick={() => setPasswordModalOpen(true)}>
                Сменить пароль
              </Button>
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Тариф">{user.tier}</Descriptions.Item>
          <Descriptions.Item label="Создан">{user.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>

      <ChangeEmailModal
        open={emailModalOpen}
        currentEmail={user.email}
        onClose={() => setEmailModalOpen(false)}
        onSuccess={() => {
          void refreshProfile();
          message.success('Email изменён');
        }}
      />
      <ChangePasswordModal
        open={passwordModalOpen}
        onClose={() => setPasswordModalOpen(false)}
        onSuccess={() => message.success('Пароль изменён')}
      />
    </>
  );
}
