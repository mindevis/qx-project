import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Routes, Route } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { LauncherLinkPage } from './LauncherLinkPage';
import { saveTokens } from '@/api/client';

function requestUrl(input: RequestInfo | URL): string {
  return typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.href
      : input.url;
}

describe('LauncherLinkPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
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

  it('links device as guest', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({
          status: 'linked',
          guest_token: 'guest-token',
          owner_type: 'guest',
        }),
        { status: 200 },
      ),
    );

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-1',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Продолжить как гость/ })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Продолжить как гость/ }));

    await waitFor(() => expect(screen.getByText('Устройство связано')).toBeInTheDocument());
  });

  it('links device as authenticated user', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
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
      }
      return Promise.resolve(
        new Response(JSON.stringify({ status: 'linked', owner_type: 'user' }), { status: 200 }),
      );
    });

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-2',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Связать устройство/ })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Связать устройство/ }));
    await waitFor(() => expect(screen.getByText('Устройство связано')).toBeInTheDocument());
  });

  it('links device with optional user code', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({
          status: 'linked',
          guest_token: 'guest-token',
          guest_expires_in: 3600,
          owner_type: 'guest',
        }),
        { status: 200 },
      ),
    );

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-code',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Продолжить как гость/ })).toBeInTheDocument(),
    );
    await user.type(screen.getByLabelText('Код с экрана лаунчера (необязательно)'), 'ABCD-1234');
    await user.click(screen.getByRole('button', { name: /Продолжить как гость/ }));
    await waitFor(() => expect(screen.getByText('Устройство связано')).toBeInTheDocument());
  });

  it('shows generic link error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockRejectedValue('boom');

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-4',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Продолжить как гость/ })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Продолжить как гость/ }));
    await waitFor(() =>
      expect(screen.getByText('Не удалось связать устройство')).toBeInTheDocument(),
    );
  });

  it('opens auth modal for guests who want to log in', async () => {
    const user = userEvent.setup({ delay: null });
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
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'X', message: 'link failed' } }), {
        status: 500,
        statusText: 'Error',
      }),
    );

    renderWithProviders(
      <Routes>
        <Route path="/launcher/link" element={<LauncherLinkPage />} />
      </Routes>,
      '/launcher/link?device=dev-3',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Продолжить как гость/ })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Продолжить как гость/ }));
    await waitFor(() => expect(screen.getByText('link failed')).toBeInTheDocument());
  });
});
