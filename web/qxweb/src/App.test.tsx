import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '@testing-library/react';
import { AuthProvider } from '@/auth/AuthContext';
import { BackendStatusProvider } from '@/backend/BackendStatusContext';
import { I18nProvider } from '@/i18n/I18nContext';
import { ThemeProvider } from '@/theme/ThemeContext';
import App from './App';
import { saveTokens } from '@/api/client';

vi.mock('skinview3d', async () => {
  const { skinview3dMock } = await import('@/test/skinview3d-mock');
  return skinview3dMock;
});

function renderApp(path = '/') {
  window.history.pushState({}, '', path);
  return render(
    <I18nProvider>
      <ThemeProvider>
        <BackendStatusProvider pollIntervalMs={50}>
          <AuthProvider>
            <App />
          </AuthProvider>
        </BackendStatusProvider>
      </ThemeProvider>
    </I18nProvider>,
  );
}

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders home route', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok' }), { status: 200 }),
    );
    renderApp('/');
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
        'Единая экосистема для Minecraft',
      ),
    );
  });

  it('renders launcher and servers pages', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );
    renderApp('/launcher');
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Войти в аккаунт' })).toBeInTheDocument(),
    );

    renderApp('/servers');
    await waitFor(() =>
      expect(screen.getByText('Управление серверами доступно после входа.')).toBeInTheDocument(),
    );
  });

  it('redirects unknown paths to home', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok' }), { status: 200 }),
    );
    renderApp('/unknown-route');
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
        'Единая экосистема для Minecraft',
      ),
    );
  });

  it('opens auth modal from legacy auth routes', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok' }), { status: 200 }),
    );
    renderApp('/auth/register');
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
    expect(screen.getByRole('tab', { name: 'Регистрация' })).toHaveAttribute('aria-selected', 'true');
  });

  it('lazy-loads profile, launcher link, and monitoring routes', async () => {
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
      return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    });

    renderApp('/profile');
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
        'Единая экосистема для Minecraft',
      ),
    );

    renderApp('/launcher/link');
    await waitFor(() =>
      expect(screen.getByText(/Не указан идентификатор устройства/)).toBeInTheDocument(),
    );

    renderApp('/monitoring');
    await waitFor(() =>
      expect(screen.getByText('Пока нет серверов в мониторинге')).toBeInTheDocument(),
    );

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
      return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    });

    renderApp('/skins');
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Скины' })).toBeInTheDocument());
  });
});
