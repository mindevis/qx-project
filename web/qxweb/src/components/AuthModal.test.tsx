import type { ComponentProps } from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { renderWithTheme } from '@/test/test-utils';
import { AuthModal } from './AuthModal';
import * as BackendStatus from '@/backend/BackendStatusContext';

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
            returnTo="/"
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
    vi.mocked(BackendStatus.useBackendStatus).mockReturnValue({ available: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('disables submit when backend is unavailable', async () => {
    vi.mocked(BackendStatus.useBackendStatus).mockReturnValue({ available: false });

    renderAuthModal();

    expect(screen.getByRole('button', { name: 'Войти' })).toBeDisabled();
    expect(
      screen.getByText('Сервер недоступен. Не удаётся связаться с API.'),
    ).toBeInTheDocument();
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

  it('shows backend unavailable on login network error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValueOnce(new TypeError('Failed to fetch'));

    renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() =>
      expect(
        screen.getByText('Сервер недоступен. Не удаётся связаться с API.'),
      ).toBeInTheDocument(),
    );
  });

  it('shows backend unavailable on bad gateway', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response('bad gateway', { status: 502, statusText: 'Bad Gateway' }),
    );

    renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() =>
      expect(
        screen.getByText('Сервер недоступен. Не удаётся связаться с API.'),
      ).toBeInTheDocument(),
    );
  });

  it('shows login error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: 'AUTH', message: 'bad login' } }), {
        status: 401,
        statusText: 'Unauthorized',
      }),
    );

    renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => expect(screen.getByText('bad login')).toBeInTheDocument());
  });

  it('shows generic login error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response('fail', { status: 418, statusText: 'Teapot' }),
    );

    renderAuthModal();

    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => expect(screen.getByText('Teapot')).toBeInTheDocument());
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
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response('fail', { status: 418, statusText: 'Teapot' }),
    );

    renderAuthModal({ mode: 'register' });

    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => expect(screen.getByText('Teapot')).toBeInTheDocument());
  });

  it('shows register error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: 'AUTH', message: 'register fail' } }), {
        status: 400,
        statusText: 'Bad Request',
      }),
    );

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
