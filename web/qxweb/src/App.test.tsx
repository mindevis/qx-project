import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '@testing-library/react';
import { AuthProvider } from '@/auth/AuthContext';
import { ThemeProvider } from '@/theme/ThemeContext';
import App from './App';

function renderApp(path = '/') {
  window.history.pushState({}, '', path);
  return render(
    <ThemeProvider>
      <AuthProvider>
        <App />
      </AuthProvider>
    </ThemeProvider>,
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
    renderApp('/');
    await waitFor(() => expect(screen.getByText('Единая экосистема для Minecraft')).toBeInTheDocument());
  });

  it('renders launcher and servers pages', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );
    renderApp('/launcher');
    await waitFor(() => expect(screen.getByRole('button', { name: /Скачать QXLauncher/ })).toBeInTheDocument());

    renderApp('/servers');
    await waitFor(() =>
      expect(screen.getByText('Управление серверами доступно после входа.')).toBeInTheDocument(),
    );
  });

  it('redirects unknown paths to home', async () => {
    renderApp('/unknown-route');
    await waitFor(() => expect(screen.getByText('Единая экосистема для Minecraft')).toBeInTheDocument());
  });

  it('opens auth modal from legacy auth routes', async () => {
    renderApp('/auth/register');
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
    expect(screen.getByRole('tab', { name: 'Регистрация' })).toHaveAttribute('aria-selected', 'true');
  });
});
