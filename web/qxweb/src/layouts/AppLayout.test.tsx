import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { AppLayout } from './AppLayout';
import * as BackendStatus from '@/backend/BackendStatusContext';
import { HomePage } from '@/pages/HomePage';
import { SkinsPage } from '@/pages/SkinsPage';
import { PlaceholderPage } from '@/pages/PlaceholderPage';
import { saveTokens } from '@/api/client';

vi.mock('skinview3d', async () => {
  const { skinview3dMock } = await import('@/test/skinview3d-mock');
  return skinview3dMock;
});

describe('AppLayout', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.mocked(BackendStatus.useBackendStatus).mockReturnValue({ available: true });
    Object.defineProperty(window, 'scrollY', { value: 0, configurable: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows login navigation for unauthenticated users', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('QXSystem')).toBeInTheDocument());
    expect(screen.getAllByRole('button', { name: 'Вход' }).length).toBeGreaterThan(0);
    expect(screen.getByRole('link', { name: 'Серверы' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Скины' })).toBeInTheDocument();
    expect(document.querySelector('.app-nav-new-badge')).toHaveTextContent('Новинка');
    expect(screen.getAllByRole('link', { name: 'Сообщество QXSystem в Discord' }).length).toBeGreaterThan(0);
  });

  it('disables login button when backend is unavailable', async () => {
    vi.mocked(BackendStatus.useBackendStatus).mockReturnValue({ available: false });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByRole('button', { name: 'Вход' })).toBeDisabled());
  });

  it('shows header spinner while auth is loading', () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    expect(document.querySelector('header .ant-spin')).toBeTruthy();
  });

  it('shows servers link in dark theme', async () => {
    const user = userEvent.setup({ delay: null, pointerEventsCheck: 0 });
    window.localStorage.setItem('qxweb-theme', 'dark');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockResolvedValueOnce(
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

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());
    expect(screen.getByRole('link', { name: 'Скины' })).toBeInTheDocument();
    await user.click(screen.getByRole('radio', { name: 'Светлая тема' }));
    expect(window.localStorage.getItem('qxweb-theme')).toBe('light');
  });

  it('shows authenticated navigation and logs out', async () => {
    const user = userEvent.setup();
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch)
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
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('US')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'US, Меню аккаунта' }));
    await user.click(await screen.findByText('Выйти'));
    await waitFor(() => expect(screen.queryByText('US')).not.toBeInTheDocument());
  });

  it('applies transparent home header that solidifies on scroll', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
      '/',
    );

    const header = document.querySelector('header');
    expect(header?.className).toContain('app-header--landing');
    expect(header?.className).not.toContain('app-header--scrolled');

    Object.defineProperty(window, 'scrollY', { value: 32, configurable: true });
    window.dispatchEvent(new Event('scroll'));

    await waitFor(() => expect(header?.className).toContain('app-header--scrolled'));
  });

  it('uses a transparent header on other sections that solidifies on scroll', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="launcher" element={<PlaceholderPage title="L" phase="P1" />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    const header = document.querySelector('header');
    expect(header?.className).toContain('app-header--sticky');
    expect(header?.className).toContain('app-header--landing');
    expect(header?.className).not.toContain('app-header--scrolled');

    Object.defineProperty(window, 'scrollY', { value: 32, configurable: true });
    window.dispatchEvent(new Event('scroll'));

    await waitFor(() => expect(header?.className).toContain('app-header--scrolled'));
  });

  it('navigates to skins page from header link', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation((input: RequestInfo | URL) => {
      const url =
        typeof input === 'string'
          ? input
          : input instanceof URL
            ? input.href
            : input.url;
      if (url.includes('/health')) {
        return Promise.resolve(
          new Response(JSON.stringify({ status: 'ok' }), { status: 200 }),
        );
      }
      if (url.includes('/auth/refresh')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              access_token: 'a',
              refresh_token: 'r',
              token_type: 'Bearer',
              expires_in: 3600,
            }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/cosmetics/skin-catalog')) {
        return Promise.resolve(
          new Response(JSON.stringify({ items: [] }), { status: 200 }),
        );
      }
      if (url.includes('/users/me/mojang')) {
        return Promise.resolve(new Response(JSON.stringify({ linked: false }), { status: 200 }));
      }
      if (url.includes('/users/me/cosmetics')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              skin_model: 'steve',
              has_skin: false,
              has_cape: false,
              updated_at: '2026-01-01T00:00:00Z',
            }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/users/me')) {
        return Promise.resolve(
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
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
          <Route path="skins" element={<SkinsPage />} />
        </Route>
      </Routes>,
      '/',
    );

    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());
    await user.click(screen.getByRole('link', { name: 'Скины' }));
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: 'Скины' })).toBeInTheDocument(),
    );
  });
});
