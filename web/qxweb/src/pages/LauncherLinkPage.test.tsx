import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Routes, Route } from 'react-router-dom';
import { message } from 'antd';
import { renderWithProviders } from '@/test/test-utils';
import { LauncherLinkPage } from './LauncherLinkPage';
import { saveTokens, clearTokens } from '@/api/client';

const messageMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  loading: vi.fn(),
  destroy: vi.fn(),
}));

vi.mock('@/hooks/useMessage', () => ({
  useMessage: () => messageMocks,
}));

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
    messageMocks.success.mockReset();
    messageMocks.error.mockReset();
  });

  afterEach(async () => {
    message.destroy();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
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
    await user.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: 'Создать аккаунт' }));
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('shows expired device error', async () => {
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-expired/status')) {
        return new Response(
          JSON.stringify(mockDeviceStatus('dev-expired', { status: 'expired' })),
          { status: 200 },
        );
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-expired',
    );
    await waitFor(() => expect(screen.getByText(/истёк/i)).toBeInTheDocument());
  });

  it('shows missing device error', async () => {
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-missing/status')) {
        return new Response(JSON.stringify({ error: { message: 'not found' } }), { status: 404 });
      }
      return new Response('not found', { status: 404 });
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-missing',
    );
    await waitFor(() =>
      expect(screen.getByText(/не найдено|not found/i)).toBeInTheDocument(),
    );
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

  it('copies device id to clipboard', async () => {
    const user = userEvent.setup({ delay: null });
    const writeText = vi.fn().mockResolvedValue(undefined);
    const clipboardSpy = vi
      .spyOn(navigator, 'clipboard', 'get')
      .mockReturnValue({ writeText } as Clipboard);
    installFetchMock((url) => {
      if (url.includes('/launcher/devices/dev-copy/status')) {
        return new Response(JSON.stringify(mockDeviceStatus('dev-copy')), { status: 200 });
      }
      return new Response('not found', { status: 404 });
    });

    try {
      renderWithProviders(
        <Routes>
          <Route path="/launcher/link" element={<LauncherLinkPage />} />
        </Routes>,
        '/launcher/link?device=dev-copy',
      );

      await waitFor(() => expect(screen.getByText('dev-copy')).toBeInTheDocument());
      await user.click(screen.getByRole('button', { name: /Копировать ID/i }));
      await waitFor(() => expect(writeText).toHaveBeenCalledWith('dev-copy'));
      expect(messageMocks.success).toHaveBeenCalled();
    } finally {
      clipboardSpy.mockRestore();
      await act(async () => {
        await new Promise((resolve) => setTimeout(resolve, 0));
      });
    }
  });
});
