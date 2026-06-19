import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { Routes, Route } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { AppLayout } from '@/layouts/AppLayout';
import { HomePage } from './HomePage';
import { LauncherPage } from './LauncherPage';
import { ProfilePage } from './ProfilePage';
import { PlaceholderPage } from './PlaceholderPage';
import { clearGuestSession, clearTokens, saveTokens } from '@/api/client';

function requestUrl(input: RequestInfo | URL): string {
  return typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.href
      : input.url;
}

function emptyProfilesResponse() {
  return new Response(JSON.stringify({ items: [] }), { status: 200 });
}

function meResponse() {
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

function mockLauncherFetch(
  handler: (url: string, init?: RequestInit) => Response | Promise<Response> | null | undefined,
) {
  return (input: RequestInfo | URL, init?: RequestInit) => {
    const url = requestUrl(input);
    const custom = handler(url, init);
    if (custom instanceof Promise) {
      return custom;
    }
    if (custom) {
      return Promise.resolve(custom);
    }
    if (url.includes('/launcher/profiles') && init?.method !== 'POST' && init?.method !== 'DELETE') {
      return Promise.resolve(emptyProfilesResponse());
    }
    if (url.includes('/users/me/launcher-device')) {
      return Promise.resolve(new Response(JSON.stringify({ linked: false }), { status: 200 }));
    }
    return Promise.resolve(meResponse());
  };
}

async function openAuthModal(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: 'Вход' }));
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
}

describe('pages', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    clearTokens();
    clearGuestSession();
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
    expect(screen.getByRole('button', { name: /Скачать QXLauncher/ })).toBeInTheDocument();
  });

  it('opens auth modal from guest launcher alert', async () => {
    const user = userEvent.setup({ delay: null });
    const { saveGuestSession } = await import('@/api/client');
    saveGuestSession({ guest_token: 'guest', expires_in: 3600 });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Гостевой режим')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Войти' }));
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
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

    await user.click(screen.getByRole('button', { name: 'Войти' }));
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
    expect(screen.getByRole('button', { name: 'Войти' })).toBeInTheDocument();
  });

  it('renders launcher page for authenticated users', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
  });

  it('shows launcher instance errors', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    let items: (typeof instance)[] = [instance];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
      if (url.includes('/instances') && init?.method === 'POST') {
        return new Response(JSON.stringify({ error: { code: 'X', message: 'create failed' } }), {
          status: 400,
          statusText: 'Bad Request',
        });
      }
      if (url.includes('/instances') && init?.method === 'DELETE') {
        return new Response(JSON.stringify({ error: { code: 'X', message: 'delete failed' } }), {
          status: 500,
          statusText: 'Error',
        });
      }
      if (url.includes('/instances')) {
        if (items.length === 0) {
          return new Response(JSON.stringify({ error: { code: 'X', message: 'list failed' } }), {
            status: 500,
            statusText: 'Error',
          });
        }
        return new Response(JSON.stringify({ items }), { status: 200 });
      }
      return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Создать/ }));
    await user.type(screen.getByLabelText('Название'), 'New');
    await user.click(screen.getByRole('button', { name: 'Создать Vanilla' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('create failed'));

    await user.click(screen.getByRole('button', { name: 'delete' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('delete failed'));

    errorSpy.mockRestore();
  });

  it('shows generic launcher errors for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/launcher/profiles')) {
        return Promise.resolve(emptyProfilesResponse());
      }
      if (url.includes('/instances') && init?.method === 'POST') {
        return Promise.reject('boom');
      }
      if (url.includes('/instances') && init?.method === 'DELETE') {
        return Promise.reject('boom');
      }
      if (url.includes('/instances')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [instance] }), { status: 200 }));
      }
      return Promise.resolve(meResponse());
    });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Создать/ }));
    await user.type(screen.getByLabelText('Название'), 'X');
    await user.click(screen.getByRole('button', { name: 'Создать Vanilla' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось создать инстанс'));

    await user.click(screen.getByRole('button', { name: 'delete' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось удалить'));
    errorSpy.mockRestore();
  });

  it('handles guest launcher access and malformed instance list', async () => {
    const { saveGuestSession } = await import('@/api/client');
    saveGuestSession({ guest_token: 'guest', expires_in: 3600 });
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/launcher/profiles')) {
        return Promise.resolve(emptyProfilesResponse());
      }
      if (url.includes('/instances')) {
        return Promise.resolve(new Response(JSON.stringify({}), { status: 200 }));
      }
      return Promise.resolve(emptyProfilesResponse());
    });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
  });

  it('shows error when instances fail to load', async () => {
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/launcher/profiles')) {
        return Promise.resolve(emptyProfilesResponse());
      }
      if (url.includes('/instances')) {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { code: 'X', message: 'list failed' } }), {
            status: 500,
            statusText: 'Error',
          }),
        );
      }
      return Promise.resolve(meResponse());
    });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith('Не удалось загрузить инстансы'),
    );
    errorSpy.mockRestore();
  });

  it('creates and deletes launcher instance', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');
    const infoSpy = vi.spyOn(message, 'info');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    let items: (typeof instance)[] = [];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
      if (url.includes('/instances') && init?.method === 'POST') {
        items = [instance];
        return new Response(JSON.stringify(instance), { status: 201 });
      }
      if (url.includes('/instances') && init?.method === 'DELETE') {
        items = [];
        return new Response(null, { status: 204 });
      }
      if (url.includes('/instances')) {
        return new Response(JSON.stringify({ items }), { status: 200 });
      }
      return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Создать/ }));
    await user.type(screen.getByLabelText('Название'), 'Survival');
    await user.click(screen.getByRole('button', { name: 'Создать Vanilla' }));

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(successSpy).toHaveBeenCalledWith('Инстанс создан');
    await waitFor(() =>
      expect(infoSpy).toHaveBeenCalledWith(
        'Создайте offline-профиль с ником или играйте с Player по умолчанию',
      ),
    );
    await waitFor(() => expect(screen.getByText('Новый offline-профиль')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Close' }));

    await user.click(screen.getByRole('button', { name: /Создать/ }));
    await waitFor(() => expect(screen.getByText('Новый инстанс')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Close' }));

    await user.click(screen.getByRole('button', { name: 'delete' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    expect(successSpy).toHaveBeenCalledWith('Инстанс удалён');
    successSpy.mockRestore();
    infoSpy.mockRestore();
  });

  it('launches with default player when no offline profile', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = vi.spyOn(message, 'info');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-1',
              status: 'queued',
              instance_id: instance.id,
              expires_at: new Date().toISOString(),
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-1')) {
          return new Response(
            JSON.stringify({
              id: 'lr-1',
              status: 'completed',
              instance_id: instance.id,
              expires_at: new Date().toISOString(),
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: /Играть/ })).toBeEnabled();
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() =>
      expect(infoSpy).toHaveBeenCalledWith(
        'Ник Player (по умолчанию). Создайте профиль выше для своего ника.',
      ),
    );
    infoSpy.mockRestore();
  });

  it('shows linked device alert and download info', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = vi.spyOn(message, 'info');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(
            JSON.stringify({ linked: true, device_id: 'dev-99', status: 'linked' }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
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
      expect(screen.getByText(/QXLauncher связан \(dev-99\)/)).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(infoSpy).toHaveBeenCalled();
    infoSpy.mockRestore();
  });

  it('unlinks launcher device from alert', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(
            JSON.stringify({ linked: true, device_id: 'dev-99', status: 'linked' }),
            { status: 200 },
          );
        }
        if (url.includes('/launcher/devices/unlink') && init?.method === 'POST') {
          return new Response(JSON.stringify({ status: 'pending_link' }), { status: 200 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
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
      expect(screen.getByText(/QXLauncher связан \(dev-99\)/)).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Отвязать/ }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('QXLauncher отвязан'));
    successSpy.mockRestore();
  });

  it('shows unlink error message', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(
            JSON.stringify({ linked: true, device_id: 'dev-99', status: 'linked' }),
            { status: 200 },
          );
        }
        if (url.includes('/launcher/devices/unlink') && init?.method === 'POST') {
          return new Response(JSON.stringify({ error: { message: 'unlink failed' } }), {
            status: 500,
          });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
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
      expect(screen.getByText(/QXLauncher связан \(dev-99\)/)).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Отвязать/ }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('unlink failed'));
    errorSpy.mockRestore();
  });

  it('shows generic unlink error for non-Error rejection', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    const { api } = await import('@/api/client');
    const unlinkSpy = vi.spyOn(api, 'unlinkDevice').mockRejectedValue('raw');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(
            JSON.stringify({ linked: true, device_id: 'dev-99', status: 'linked' }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
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
      expect(screen.getByText(/QXLauncher связан \(dev-99\)/)).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Отвязать/ }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith('Не удалось отвязать устройство'),
    );
    errorSpy.mockRestore();
    unlinkSpy.mockRestore();
  });

  it('defaults linked device status when omitted', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(JSON.stringify({ linked: true, device_id: 'dev-42' }), {
            status: 200,
          });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
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
      expect(screen.getByText(/QXLauncher связан \(dev-42\)/)).toBeInTheDocument(),
    );
  });

  it('handles profiles list without items field', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({}), { status: 200 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
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
      expect(
        screen.getByText('Нет профилей — можно играть как Player или добавьте свой ник'),
      ).toBeInTheDocument(),
    );
  });

  it('opens launcher download URL when configured', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    vi.stubEnv('VITE_LAUNCHER_DOWNLOAD_URL', 'https://releases.example/qx-launcher.exe');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    const user = userEvent.setup({ delay: null });
    await waitFor(() => expect(screen.getByRole('button', { name: /Скачать QXLauncher/ })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(openSpy).toHaveBeenCalledWith(
      'https://releases.example/qx-launcher.exe',
      '_blank',
      'noopener,noreferrer',
    );
    openSpy.mockRestore();
    vi.unstubAllEnvs();
  });

  it('shows error when profiles fail to load', async () => {
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/launcher/profiles')) {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { code: 'X', message: 'profiles failed' } }), {
            status: 500,
          }),
        );
      }
      if (url.includes('/instances')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
      }
      return Promise.resolve(meResponse());
    });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith('Не удалось загрузить профили'),
    );
    errorSpy.mockRestore();
  });

  it('shows generic play error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return Promise.reject('launch boom');
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось запустить игру'));
    errorSpy.mockRestore();
  });

  it('skips profile modal after create when profiles exist', async () => {
    const user = userEvent.setup({ delay: null });
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    let items: (typeof instance)[] = [];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances') && init?.method === 'POST') {
          items = [instance];
          return new Response(JSON.stringify(instance), { status: 201 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [profile] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByRole('heading', { name: 'Steve' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Создать/ }));
    await user.type(screen.getByLabelText('Название'), 'Survival');
    await user.click(screen.getByRole('button', { name: 'Создать Vanilla' }));
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.queryByText('Новый offline-профиль')).not.toBeInTheDocument();
  });

  it('reports failed launch and deletes profile', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    const successSpy = vi.spyOn(message, 'success');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    let profiles: (typeof profile)[] = [];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles') && init?.method === 'POST') {
          profiles = [profile];
          return new Response(JSON.stringify(profile), { status: 201 });
        }
        if (url.includes('/launcher/profiles') && init?.method === 'DELETE') {
          profiles = [];
          return new Response(null, { status: 204 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: profiles }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-fail',
              status: 'queued',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-fail')) {
          return new Response(
            JSON.stringify({
              id: 'lr-fail',
              status: 'failed',
              error_code: 'JAVA_MISSING',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Добавить/ }));
    await user.type(screen.getByLabelText('Никнейм'), 'Steve');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Steve' })).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('JAVA_MISSING'));

    const deleteButtons = screen.getAllByRole('button', { name: 'delete' });
    await user.click(deleteButtons[0]!);
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Профиль удалён'));
    errorSpy.mockRestore();
    successSpy.mockRestore();
  });

  it('creates profile and launches instance', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    let profiles: (typeof profile)[] = [];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles') && init?.method === 'POST') {
          profiles = [profile];
          return new Response(JSON.stringify(profile), { status: 201 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: profiles }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-1',
              status: 'queued',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-1')) {
          return new Response(
            JSON.stringify({
              id: 'lr-1',
              status: 'completed',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Добавить/ }));
    await user.type(screen.getByLabelText('Никнейм'), 'Steve');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Steve' })).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Игра запущена'));
    successSpy.mockRestore();
  });

  it('warns when launch request expires', async () => {
    const user = userEvent.setup({ delay: null });
    const warningSpy = vi.spyOn(message, 'warning');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-exp',
              status: 'queued',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-exp')) {
          return new Response(
            JSON.stringify({
              id: 'lr-exp',
              status: 'expired',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(warningSpy).toHaveBeenCalled());
    warningSpy.mockRestore();
  });

  it('reports intermediate launch statuses while polling', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = vi.spyOn(message, 'info');
    const successSpy = vi.spyOn(message, 'success');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    let poll = 0;
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [profile] }), { status: 200 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-run',
              status: 'queued',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-run')) {
          poll += 1;
          const status =
            poll === 1 ? 'queued' : poll === 2 ? 'dispatched' : poll === 3 ? 'running' : 'completed';
          return new Response(
            JSON.stringify({
              id: 'lr-run',
              status,
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(
      () => expect(infoSpy).toHaveBeenCalledWith('Minecraft запускается…', 2),
      { timeout: 8000 },
    );
    expect(infoSpy).toHaveBeenCalledWith('Запрос в очереди…', 2);
    expect(infoSpy).toHaveBeenCalledWith('QXLauncher получил запрос…', 2);
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Игра запущена'), {
      timeout: 8000,
    });
    infoSpy.mockRestore();
    successSpy.mockRestore();
  });

  it('ignores linked device load failures', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/launcher-device')) {
          return Promise.reject(new Error('device unavailable'));
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
  });

  it('shows generic profile errors for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles') && init?.method === 'POST') {
          return Promise.reject('profile boom');
        }
        if (url.includes('/launcher/profiles') && init?.method === 'DELETE') {
          return Promise.reject('delete boom');
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [profile] }), { status: 200 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByRole('heading', { name: 'Steve' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Добавить/ }));
    await user.type(screen.getByLabelText('Никнейм'), 'Alex');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось создать профиль'));
    errorSpy.mockRestore();
  });

  it('shows delete profile api error', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles/prof-1') && init?.method === 'DELETE') {
          return new Response(
            JSON.stringify({ error: { code: 'X', message: 'delete denied' } }),
            { status: 403 },
          );
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [profile] }), { status: 200 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByRole('heading', { name: 'Steve' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'delete' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('delete denied'));
    errorSpy.mockRestore();
  });

  it('clears selected profile after deleting it', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles') && init?.method === 'DELETE') {
          return new Response(null, { status: 204 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [profile] }), { status: 200 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByRole('heading', { name: 'Steve' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'delete' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Профиль удалён'));
    successSpy.mockRestore();
  });

  it('opens profile modal when profiles response omits items', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = vi.spyOn(message, 'info');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    let items: (typeof instance)[] = [];
    let profileCalls = 0;
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances') && init?.method === 'POST') {
          items = [instance];
          return new Response(JSON.stringify(instance), { status: 201 });
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          profileCalls += 1;
          if (profileCalls > 1) {
            return new Response(JSON.stringify({}), { status: 200 });
          }
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Создать/ }));
    await user.type(screen.getByLabelText('Название'), 'Survival');
    await user.click(screen.getByRole('button', { name: 'Создать Vanilla' }));
    await waitFor(() =>
      expect(infoSpy).toHaveBeenCalledWith(
        'Создайте offline-профиль с ником или играйте с Player по умолчанию',
      ),
    );
    infoSpy.mockRestore();
  });

  it('shows generic failed launch error without error code', async () => {
    const user = userEvent.setup({ delay: null });
    const errorSpy = vi.spyOn(message, 'error');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-fail2',
              status: 'queued',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-fail2')) {
          return new Response(
            JSON.stringify({
              id: 'lr-fail2',
              status: 'failed',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Ошибка запуска'));
    errorSpy.mockRestore();
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
