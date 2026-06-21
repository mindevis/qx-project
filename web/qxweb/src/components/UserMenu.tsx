import { useNavigate } from 'react-router-dom';
import { Avatar, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { LogoutOutlined, UserOutlined } from '@ant-design/icons';
import type { UserProfile } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';

type UserMenuProps = {
  user: UserProfile;
  onLogout: () => Promise<void>;
};

export function emailInitials(email: string): string {
  return email.slice(0, 2).toUpperCase();
}

export function UserMenu({ user, onLogout }: UserMenuProps) {
  const navigate = useNavigate();
  const { t } = useI18n();

  const items: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: t('common.profile'),
      onClick: () => navigate('/profile'),
    },
    { type: 'divider' },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('common.logout'),
      onClick: () => {
        void onLogout();
      },
    },
  ];

  return (
    <Dropdown menu={{ items }} placement="bottomRight" trigger={['click']}>
      <span
        role="button"
        tabIndex={0}
        aria-label={t('common.accountMenu')}
        style={{ cursor: 'pointer', lineHeight: 0 }}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            e.currentTarget.click();
          }
        }}
      >
        <Avatar src={user.avatar_url}>{!user.avatar_url && emailInitials(user.email)}</Avatar>
      </span>
    </Dropdown>
  );
}
