import { useNavigate } from 'react-router-dom';
import { Avatar, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { LogoutOutlined, UserOutlined } from '@ant-design/icons';
import type { UserProfile } from '@/api/client';

type UserMenuProps = {
  user: UserProfile;
  onLogout: () => Promise<void>;
};

export function emailInitials(email: string): string {
  return email.slice(0, 2).toUpperCase();
}

export function UserMenu({ user, onLogout }: UserMenuProps) {
  const navigate = useNavigate();

  const items: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: 'Профиль',
      onClick: () => navigate('/profile'),
    },
    { type: 'divider' },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Выйти',
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
        aria-label="Меню аккаунта"
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
