import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { emailInitials, UserMenu } from './UserMenu';

const user = {
  id: '1',
  email: 'user@test.com',
  tier: 'free',
  created_at: 'now',
};

function renderMenu(props: Partial<ComponentProps<typeof UserMenu>> = {}) {
  const onLogout = vi.fn().mockResolvedValue(undefined);
  render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route
          path="/"
          element={<UserMenu user={user} onLogout={onLogout} {...props} />}
        />
        <Route path="/profile" element={<div>Profile page</div>} />
      </Routes>
    </MemoryRouter>,
  );
  return { onLogout };
}

describe('emailInitials', () => {
  it('returns first two letters uppercased', () => {
    expect(emailInitials('user@test.com')).toBe('US');
    expect(emailInitials('ab@example.com')).toBe('AB');
  });
});

describe('UserMenu', () => {
  it('navigates to profile', async () => {
    const clicker = userEvent.setup({ delay: null });
    renderMenu();

    await clicker.click(screen.getByRole('button', { name: 'Меню аккаунта' }));
    await clicker.click(screen.getByText('Профиль'));
    expect(screen.getByText('Profile page')).toBeInTheDocument();
  });

  it('logs out from menu', async () => {
    const clicker = userEvent.setup({ delay: null });
    const { onLogout } = renderMenu();

    await clicker.click(screen.getByRole('button', { name: 'Меню аккаунта' }));
    await clicker.click(screen.getByText('Выйти'));
    expect(onLogout).toHaveBeenCalled();
  });

  it('opens menu on Enter and Space', async () => {
    const clicker = userEvent.setup({ delay: null });
    renderMenu();

    const trigger = screen.getByRole('button', { name: 'Меню аккаунта' });
    trigger.focus();
    await clicker.keyboard('{Enter}');
    expect(screen.getByText('Профиль')).toBeInTheDocument();

    await clicker.keyboard(' ');
    expect(screen.getByText('Выйти')).toBeInTheDocument();
  });

  it('ignores unrelated keys on account menu trigger', async () => {
    const clicker = userEvent.setup({ delay: null });
    renderMenu();

    const trigger = screen.getByRole('button', { name: 'Меню аккаунта' });
    trigger.focus();
    await clicker.keyboard('{ArrowDown}');
    expect(screen.queryByText('Профиль')).not.toBeInTheDocument();
  });

  it('renders avatar image when avatar_url is set', () => {
    renderMenu({
      user: { ...user, avatar_url: 'https://example.com/a.png' },
    });

    expect(screen.getByRole('img')).toHaveAttribute('src', 'https://example.com/a.png');
  });
});
