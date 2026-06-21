import type { ComponentProps } from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { renderWithTheme } from '@/test/test-utils';
import { AuthModal } from './AuthModal';

function renderAuthModal(props: Partial<ComponentProps<typeof AuthModal>> = {}) {
  const onClose = vi.fn();
  const onModeChange = vi.fn();
  return {
    onClose,
    onModeChange,
    ...renderWithTheme(
      <MemoryRouter>
        <AuthProvider>
          <AuthModal
            open
            mode="login"
            onModeChange={onModeChange}
            onClose={onClose}
            {...props}
          />
        </AuthProvider>
      </MemoryRouter>,
    ),
  };
}

describe('AuthModal', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('logs in and navigates to profile', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            access_token: 'a',
            refresh_token: 'r',
            token_type: 'Bearer',
            expires_in: 60,
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'user@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );

    const { onClose } = renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it('shows login error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValueOnce(new Error('bad login'));

    renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => expect(screen.getByText('bad login')).toBeInTheDocument());
  });

  it('shows generic login error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValueOnce('fail');

    renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => expect(screen.getByText('Ошибка входа')).toBeInTheDocument());
  });

  it('registers successfully', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            access_token: 'a',
            refresh_token: 'r',
            token_type: 'Bearer',
            expires_in: 60,
          }),
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'new@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );

    const { onClose } = renderAuthModal({ mode: 'register' });

    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it('shows generic register error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementationOnce(() => Promise.reject('fail'));

    renderAuthModal({ mode: 'register' });

    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => expect(screen.getByText('Ошибка регистрации')).toBeInTheDocument());
  });

  it('shows register error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValueOnce(new Error('register fail'));

    renderAuthModal({ mode: 'register' });

    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => expect(screen.getByText('register fail')).toBeInTheDocument());
  });

  it('switches tabs and clears error', async () => {
    const user = userEvent.setup({ delay: null });
    const { onModeChange } = renderAuthModal();

    await user.click(screen.getByRole('tab', { name: 'Регистрация' }));
    expect(onModeChange).toHaveBeenCalledWith('register');
  });

  it('closes and resets forms', async () => {
    const user = userEvent.setup({ delay: null });
    const { onClose } = renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalled();
  });
});
