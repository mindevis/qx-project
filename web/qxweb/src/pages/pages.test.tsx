import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Routes, Route } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { AppLayout } from '@/layouts/AppLayout';
import { HomePage } from './HomePage';
import { LauncherPage } from './LauncherPage';
import { ProfilePage } from './ProfilePage';
import { PlaceholderPage } from './PlaceholderPage';
import { saveTokens } from '@/api/client';

async function openAuthModal(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: 'Вход' }));
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
}

describe('pages', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders home as description page', () => {
    renderWithProviders(<HomePage />);
    expect(screen.getByText('Единая экосистема для Minecraft')).toBeInTheDocument();
    expect(screen.getByText('QXWeb')).toBeInTheDocument();
    expect(screen.getByText('QXLauncher')).toBeInTheDocument();
    expect(screen.getByText('QXAgent')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Скачать QXLauncher/ })).not.toBeInTheDocument();
  });

  it('opens auth modal from launcher page for guests', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await user.click(screen.getByRole('button', { name: 'Войти, чтобы открыть лаунчер' }));
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('renders launcher page for guests', () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    expect(screen.getByRole('button', { name: /Скачать QXLauncher/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Войти, чтобы открыть лаунчер' })).toBeInTheDocument();
  });

  it('renders launcher page for authenticated users', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({
          id: '1',
          email: 'u@test.com',
          tier: 'free',
          created_at: 'now',
        }),
        { status: 200 },
      ),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Открыть лаунчер/ })).toBeInTheDocument(),
    );
  });

  it('logs in successfully', async () => {
    const user = userEvent.setup();
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
            email: 'u@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Routes>,
      '/',
    );

    await openAuthModal(user);
    await user.type(screen.getByLabelText('Email'), 'u@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => {
      expect(screen.getAllByText('u@test.com').length).toBeGreaterThan(0);
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  it('shows login error message', async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockRejectedValueOnce(new Error('invalid credentials'));

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
        </Route>
      </Routes>,
      '/',
    );

    await openAuthModal(user);
    await user.type(screen.getByLabelText('Email'), 'bad@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitFor(() => expect(screen.getByText('invalid credentials')).toBeInTheDocument());
  });

  it('shows generic login error for non-error throws', async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockRejectedValueOnce('fail');

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
        </Route>
      </Routes>,
      '/',
    );

    await openAuthModal(user);
    await user.type(screen.getByLabelText('Email'), 'bad@test.com');
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

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Routes>,
      '/',
    );

    await openAuthModal(user);
    await user.click(screen.getByRole('tab', { name: 'Регистрация' }));
    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => {
      expect(screen.getAllByText('new@test.com').length).toBeGreaterThan(0);
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  it('shows register error message', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValueOnce(new Error('boom'));

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
        </Route>
      </Routes>,
      '/',
    );

    await openAuthModal(user);
    await user.click(screen.getByRole('tab', { name: 'Регистрация' }));
    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument());
  });

  it('shows generic register error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValueOnce('fail');

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
        </Route>
      </Routes>,
      '/',
    );

    await openAuthModal(user);
    await user.click(screen.getByRole('tab', { name: 'Регистрация' }));
    await user.type(screen.getByLabelText('Email'), 'new@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.type(screen.getByLabelText('Повтор пароля'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));

    await waitFor(() => expect(screen.getByText('Ошибка регистрации')).toBeInTheDocument());
  });

  it('shows profile spinner and content', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      () =>
        new Promise((resolve) => {
          setTimeout(
            () =>
              resolve(
                new Response(
                  JSON.stringify({
                    id: '1',
                    email: 'u@test.com',
                    tier: 'free',
                    created_at: 'now',
                  }),
                  { status: 200 },
                ),
              ),
            50,
          );
        }),
    );

    renderWithProviders(<ProfilePage />, '/profile');
    expect(document.querySelector('.ant-spin')).toBeTruthy();
    await waitFor(() => expect(screen.getByRole('button', { name: 'Сменить пароль' })).toBeInTheDocument());
    expect(screen.getByText('u@test.com')).toBeInTheDocument();
  });

  it('redirects unauthenticated profile', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Routes>,
      '/profile',
    );

    await waitFor(() => expect(screen.getByRole('button', { name: 'Вход' })).toBeInTheDocument());
  });

  it('renders placeholder page', () => {
    renderWithProviders(<PlaceholderPage title="Лаунчер" phase="Phase 1" />);
    expect(screen.getByText('Лаунчер')).toBeInTheDocument();
    expect(screen.getByText(/Phase 1/)).toBeInTheDocument();
  });
});
