import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '@testing-library/react';
import { AuthProvider } from '@/auth/AuthContext';
import { BackendStatusProvider } from '@/backend/BackendStatusContext';
import { I18nProvider } from '@/i18n/I18nContext';
import { ThemeProvider } from '@/theme/ThemeContext';
import App from './App';

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
});
