import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Routes, Route } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { LauncherLinkPage } from './LauncherLinkPage';
import { saveTokens, clearTokens } from '@/api/client';

function requestUrl(input: RequestInfo | URL): string {
  return typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.href
      : input.url;
}

function mockDeviceStatus(
  deviceId: string,
  overrides: Record<string, unknown> = {},
) {
  return {
    status: 'pending_link',
    device_id: deviceId,
    hostname: 'DESKTOP-TEST',
    os: 'windows',
    launcher_version: '0.1.0',
    link_expires_at: '2026-12-31T12:00:00Z',
    last_seen_at: '2026-06-10T12:00:00Z',
    ...overrides,
  };
}

function installFetchMock(handler: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.mocked(fetch).mockImplementation((input, init) => {
    const url = requestUrl(input);
    return Promise.resolve(handler(url, init));
  });
}

function saveAuthTokens() {
  saveTokens({
    access_token: 'a',
    refresh_token: 'r',
    token_type: 'Bearer',
    expires_in: 60,
  });
}

describe('LauncherLinkPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    clearTokens();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clearTokens();
  });

  it('shows error without device param', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link',
    );

    await waitFor(() =>
      expect(screen.getByText(/Не указан идентификатор устройства/)).toBeInTheDocument(),
    );
  });

  it('shows full device info from status endpoint', async () => {
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-1/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-1')), { status: 200 });
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-1',
    );

    await waitFor(() => {
      expect(screen.getByText('dev-1')).toBeInTheDocument();
      expect(screen.getByText('DESKTOP-TEST')).toBeInTheDocument();
      expect(screen.getByText('Windows')).toBeInTheDocument();
      expect(screen.getByText('0.1.0')).toBeInTheDocument();
      expect(screen.getByText('Ожидает привязки')).toBeInTheDocument();
    });
  });

  it('shows sign-in prompt when unauthenticated', async () => {
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-1/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-1')), { status: 200 });
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-1',
    );

    await waitFor(() => expect(screen.getByText('Нужен аккаунт')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: /Связать устройство/ })).not.toBeInTheDocument();
  });

  it('links device as authenticated user', async () => {
    const user = userEvent.setup({ delay: null });
    saveAuthTokens();
    installFetchMock((url, init) => {
      const urlStr = url;
      if (urlStr.includes('/users/me')) {
        return new Response(
          JSON.stringify({
            id: '1',
            email: 'u@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        );
      }
      if (urlStr.includes('/launcher/devices/dev-2/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-2')), { status: 200 });
      }
      if (urlStr.includes('/launcher/devices/link') && init?.method === 'POST') {
        return new Response(JSON.stringify({ status: 'linked', owner_type: 'user' }), { status: 200 });
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-2',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Связать устройство/ })).toBeEnabled(),
    );
    await user.click(screen.getByRole('button', { name: /Связать устройство/ }));
    await waitFor(() => expect(screen.getByText('Устройство связано')).toBeInTheDocument());
  });

  it('shows generic link error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    saveAuthTokens();
    installFetchMock((url, init) => {
      if (url.includes('/users/me')) {
        return new Response(
          JSON.stringify({
            id: '1',
            email: 'u@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        );
      }
      if (url.includes('/launcher/devices/dev-4/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-4')), { status: 200 });
      }
      if (url.includes('/launcher/devices/link') && init?.method === 'POST') {
        return Promise.reject('boom');
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-4',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Связать устройство/ })).toBeEnabled(),
    );
    await user.click(screen.getByRole('button', { name: /Связать устройство/ }));
    await waitFor(() =>
      expect(screen.getByText('Не удалось связать устройство')).toBeInTheDocument(),
    );
  });

  it('opens auth modal for unauthenticated users', async () => {
    const user = userEvent.setup({ delay: null });
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-login/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-login')), { status: 200 });
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-login',
    );

    await waitFor(() => expect(screen.getByRole('button', { name: 'Войти' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Войти' }));
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('shows link error', async () => {
    const user = userEvent.setup({ delay: null });
    saveAuthTokens();
    installFetchMock((url, init) => {
      if (url.includes('/users/me')) {
        return new Response(
          JSON.stringify({
            id: '1',
            email: 'u@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        );
      }
      if (url.includes('/launcher/devices/dev-3/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-3')), { status: 200 });
      }
      if (url.includes('/launcher/devices/link') && init?.method === 'POST') {
        return new Response(JSON.stringify({ error: { code: 'X', message: 'link failed' } }), {
          status: 500,
          statusText: 'Error',
        });
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-3',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Связать устройство/ })).toBeEnabled(),
    );
    await user.click(screen.getByRole('button', { name: /Связать устройство/ }));
    await waitFor(() => expect(screen.getByText('link failed')).toBeInTheDocument());
  });

  it('shows already linked state without link button', async () => {
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-linked/status')) {
        return new Response(
          JSON.stringify(mockDeviceStatus('dev-linked', { status: 'linked' })),
          { status: 200 },
        );
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-linked',
    );

    await waitFor(() => expect(screen.getByText('Устройство уже привязано')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: /Связать устройство/ })).not.toBeInTheDocument();
  });
});
