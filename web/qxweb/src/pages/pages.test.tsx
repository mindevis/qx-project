import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { Routes, Route } from 'react-router-dom';
import { renderWithProviders, waitForNoDialog } from '@/test/test-utils';
import { AppLayout } from '@/layouts/AppLayout';
import { HomePage, highlightMinecraft } from './HomePage';
import { LauncherPage } from './LauncherPage';
import { ProfilePage } from './ProfilePage';
import { PlaceholderPage } from './PlaceholderPage';
import { clearTokens, saveTokens, api } from '@/api/client';
import * as launcherDownload from '@/lib/launcherDownload';
import { LAUNCHER_INSTANCES_VIEW_STORAGE_KEY } from '@/lib/installedResourcesView';

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

function offlineProfileResponse() {
  return new Response(
    JSON.stringify({
      items: [{ id: 'prof-1', username: 'Steve', model: 'steve' }],
    }),
    { status: 200 },
  );
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

function cosmeticsResponse() {
  return new Response(
    JSON.stringify({
      skin_model: 'steve',
      has_skin: false,
      updated_at: '2026-01-01T00:00:00Z',
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
    if (url.includes('/launcher/release')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            version: '0.2.0',
            download_url: '/downloads/qx-launcher.exe',
            filename: 'qx-launcher.exe',
          }),
          { status: 200 },
        ),
      );
    }
    if (url.includes('/launcher/mc-versions')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            latest: { release: '1.21' },
            items: [
              { id: '1.21', type: 'release' },
              { id: '1.20.4', type: 'release' },
            ],
          }),
          { status: 200 },
        ),
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
    if (url.includes('/users/me/launcher-device')) {
      return Promise.resolve(new Response(JSON.stringify({ linked: false }), { status: 200 }));
    }
    if (url.includes('/users/me/cosmetics')) {
      return Promise.resolve(cosmeticsResponse());
    }
    if (url.includes('/users/me/mojang')) {
      return Promise.resolve(new Response(JSON.stringify({ linked: false }), { status: 200 }));
    }
    if (url.includes('/users/me')) {
      return Promise.resolve(meResponse());
    }
    return Promise.resolve(meResponse());
  };
}

async function openAuthModal(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: 'Вход' }));
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
}

async function expectAuthModalError(message: string) {
  await waitFor(() => {
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText(message)).toBeInTheDocument();
  });
}

async function expectLauncherProfileListed(username: string) {
  await waitFor(() => {
    const names = [...document.querySelectorAll('.launcher-profile-name')].map((el) => el.textContent ?? '');
    expect(names.some((name) => name.includes(username))).toBe(true);
  });
}

async function openCreateInstanceModal(user: ReturnType<typeof userEvent.setup>) {
  await waitFor(() => {
    const hasCreate =
      screen.queryByRole('button', { name: /Создать первый инстанс/ }) ??
      screen.queryAllByRole('button', { name: /Новый инстанс/ })[0];
    expect(hasCreate).toBeTruthy();
  });
  const createFirst = screen.queryByRole('button', { name: /Создать первый инстанс/ });
  if (createFirst) {
    await user.click(createFirst);
    return;
  }
  await user.click(screen.getAllByRole('button', { name: /Новый инстанс/ })[0]!);
}

async function openAddProfileModal(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getAllByRole('button', { name: /Добавить профиль/ })[0]!);
}

function getProfileDeleteButton() {
  return screen.getByRole('button', { name: 'Удалить профиль?' });
}

describe.sequential('pages', { timeout: 30_000 }, () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(new Response('{}', { status: 200 }))),
    );
    clearTokens();
    window.localStorage.removeItem(LAUNCHER_INSTANCES_VIEW_STORAGE_KEY);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders home as description page', () => {
    renderWithProviders(<HomePage />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'Единая экосистема для Minecraft',
    );
    expect(screen.getAllByText('QXLauncher').length).toBeGreaterThan(0);
    expect(screen.getAllByText('QXAgent').length).toBeGreaterThan(0);
    expect(screen.getByRole('heading', { name: 'MySQL и Ollama на вашем VPS' })).toBeInTheDocument();
    expect(screen.getAllByText('MySQL').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Ollama').length).toBeGreaterThan(0);
    expect(screen.getAllByRole('button', { name: /Открыть лаунчер/ }).length).toBeGreaterThan(0);
    expect(screen.getByRole('link', { name: 'Сообщество QXSystem в Discord' })).toHaveAttribute(
      'href',
      'https://discord.gg/uNtN2yAGnA',
    );
  });

  it('highlightMinecraft returns plain text when marker is absent', () => {
    expect(highlightMinecraft('No game here')).toBe('No game here');
  });

  it('shows servers CTA on home for authenticated users', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockResolvedValue(
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

    renderWithProviders(<HomePage />);

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Управление серверами' })).toBeInTheDocument(),
    );
  });

  it('opens auth modal from launcher sign-in prompt', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch(() => null),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(screen.getByText('Войдите в аккаунт, чтобы управлять инстансами и профилями.')).toBeInTheDocument(),
    );
    await user.click(screen.getAllByRole('button', { name: 'Войти в аккаунт' })[0]!);
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('opens auth modal from launcher page when unauthenticated', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await user.click(screen.getAllByRole('button', { name: 'Войти в аккаунт' })[0]!);
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('renders launcher page for unauthenticated users', () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    expect(screen.getByText('Сначала свяжите QXLauncher')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Проверить связь/ })).toBeInTheDocument();
    expect(
      screen.getByText('Войдите в аккаунт, чтобы управлять инстансами и профилями.'),
    ).toBeInTheDocument();
  });

  it('refreshes launcher access on storage and focus events', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(screen.getByText('Сначала свяжите QXLauncher')).toBeInTheDocument(),
    );

    window.dispatchEvent(new Event('storage'));
    window.dispatchEvent(new Event('focus'));

    expect(screen.getByText('Сначала свяжите QXLauncher')).toBeInTheDocument();
  });

  it('renders launcher page for authenticated users', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
  });

  it('shows launcher instance errors', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const items: (typeof instance)[] = [instance];
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await openCreateInstanceModal(user);
    await user.type(screen.getByLabelText('Название'), 'New');
    await user.click(screen.getByRole('button', { name: 'Создать инстанс' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('create failed'));

    await user.click(screen.getByRole('button', { name: 'Удалить инстанс?' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('delete failed'));

  });

  it('shows generic launcher errors for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
        if (url.includes('/instances') && init?.method === 'POST') {
          return null;
        }
        if (url.includes('/instances') && init?.method === 'DELETE') {
          return null;
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        return null;
      }),
    );
    const createSpy = vi.spyOn(api, 'createInstance').mockRejectedValueOnce('boom' as never);
    const deleteSpy = vi.spyOn(api, 'deleteInstance').mockRejectedValueOnce('boom' as never);
    const cloneSpy = vi.spyOn(api, 'cloneInstance').mockRejectedValueOnce('boom' as never);

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await openCreateInstanceModal(user);
    await user.type(screen.getByLabelText('Название'), 'X');
    await user.click(screen.getByRole('button', { name: 'Создать инстанс' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось создать инстанс'));
    createSpy.mockRestore();

    await user.click(screen.getByRole('button', { name: 'Удалить инстанс?' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось удалить'));
    deleteSpy.mockRestore();

    await user.click(screen.getByRole('button', { name: 'Клонировать инстанс?' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось клонировать инстанс'));
    cloneSpy.mockRestore();
  });

  it('shows sign-in required when unauthenticated on launcher workspace', async () => {
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/launcher/profiles')) {
        return Promise.resolve(emptyProfilesResponse());
      }
      return Promise.resolve(emptyProfilesResponse());
    });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(screen.getByText('Войдите в аккаунт, чтобы управлять инстансами и профилями.')).toBeInTheDocument(),
    );
  });

  it('shows error when instances fail to load', async () => {
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith('Не удалось загрузить инстансы'),
    );
  });

  it('creates and deletes launcher instance', async () => {
    const user = userEvent.setup({ delay: null });
  const successSpy = testMessage.success;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    await openCreateInstanceModal(user);
    await user.type(screen.getByLabelText('Название'), 'Survival');
    await user.click(screen.getByRole('button', { name: 'Создать инстанс' }));

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(successSpy).toHaveBeenCalledWith('Инстанс создан');
    expect(screen.queryByRole('dialog', { name: 'Новый профиль игрока' })).not.toBeInTheDocument();

    await openCreateInstanceModal(user);
    await waitFor(() => expect(screen.getByRole('dialog', { name: 'Новый инстанс' })).toBeInTheDocument());
    await user.click(screen.getAllByRole('button', { name: 'Close' })[0]!);
    await waitForNoDialog();

    await user.click(screen.getByRole('button', { name: 'Удалить инстанс?' }));
    await user.click(await screen.findByRole('button', { name: 'OK' }));

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    expect(successSpy).toHaveBeenCalledWith('Инстанс удалён');
  });

  it('falls back when mc versions fail to load', async () => {
  const warnSpy = testMessage.warning;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/launcher/mc-versions')) {
          return new Response('fail', { status: 500 });
        }
        if (url.includes('/instances') || url.includes('/launcher/profiles') || url.includes('/launcher/devices')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(warnSpy).toHaveBeenCalledWith('Не удалось загрузить список версий'),
    );
  });

  it('reloads create form versions when loader type changes', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances') || url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('fabric') || url.includes('meta.fabricmc')) {
          return new Response(
            JSON.stringify({
              versions: [{ version: '1.21', stable: true }],
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    await openCreateInstanceModal(user);
    const dialog = await screen.findByRole('dialog', { name: 'Новый инстанс' });
    const comboboxes = within(dialog).getAllByRole('combobox');
    await user.click(comboboxes[0]!);
    await user.click(await screen.findByText('Fabric', { selector: '.ant-select-item-option-content' }));
    await waitFor(() =>
      expect(within(dialog).getByLabelText('Версия Fabric Loader')).toBeInTheDocument(),
    );
    const mcCombobox = within(dialog).getAllByRole('combobox')[1]!;
    await user.click(mcCombobox);
    await user.click(await screen.findByText('1.20.4', { selector: '.ant-select-item-option-content' }));
    await waitFor(() =>
      expect(within(dialog).getByLabelText('Версия Fabric Loader')).toBeInTheDocument(),
    );
  });

  it('falls back when create mc versions fail for mod loader', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances') || url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('fabric') && url.includes('game')) {
          return new Response('fail', { status: 500 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await openCreateInstanceModal(user);
    const dialog = await screen.findByRole('dialog', { name: 'Новый инстанс' });
    const comboboxes = within(dialog).getAllByRole('combobox');
    await user.click(comboboxes[0]!);
    await user.click(await screen.findByText('Fabric', { selector: '.ant-select-item-option-content' }));
    await waitFor(() => expect(within(dialog).getByLabelText('Версия Minecraft')).toBeInTheDocument());
  });

  it('handles loader version fetch failure in create modal', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances') || url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('loader') && url.includes('fabric')) {
          return new Response('fail', { status: 500 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await openCreateInstanceModal(user);
    const dialog = await screen.findByRole('dialog', { name: 'Новый инстанс' });
    const comboboxes = within(dialog).getAllByRole('combobox');
    await user.click(comboboxes[0]!);
    await user.click(await screen.findByText('Fabric', { selector: '.ant-select-item-option-content' }));
    await waitFor(() =>
      expect(within(dialog).getByLabelText('Версия Fabric Loader')).toBeInTheDocument(),
    );
  });

  it('opens profile modal from hero action', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances') || url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Игрок')).toBeInTheDocument());
    await user.click(screen.getAllByRole('button', { name: /Новый профиль/ })[0]!);
    await waitFor(() =>
      expect(screen.getByRole('dialog', { name: 'Новый профиль игрока' })).toBeInTheDocument(),
    );
  });

  it('selects an offline profile for launch', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const profiles = [
      {
        id: 'prof-1',
        username: 'Steve',
        model: 'steve' as const,
        offline_uuid: 'u1',
        created_at: 't',
      },
      {
        id: 'prof-2',
        username: 'AlexName',
        model: 'alex' as const,
        offline_uuid: 'u2',
        created_at: 't',
      },
    ];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: profiles }), { status: 200 });
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await user.click(screen.getByText('AlexName'));
    await waitFor(() => {
      const selected = document.querySelector('.launcher-profile-chip--selected .launcher-profile-name');
      expect(selected).toHaveTextContent('AlexName');
    });
  });

  it('does not show play-as Player or launch without a profile', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const createLaunch = vi.fn();
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          createLaunch();
          return new Response('{}', { status: 500 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.queryByText('Играть как')).not.toBeInTheDocument();
    expect(screen.queryByText('Player')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Играть/ })).toBeDisabled();
    expect(createLaunch).not.toHaveBeenCalled();
  });

  it('shows launching status badge while polling launch request', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    let polls = 0;
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return offlineProfileResponse();
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
        if (url.includes('/launcher/launch-requests/lr-1') && init?.method !== 'POST') {
          polls += 1;
          const status = polls < 10 ? 'preparing' : 'completed';
          return new Response(
            JSON.stringify({
              id: 'lr-1',
              status,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(screen.getByText('Идёт запуск')).toBeInTheDocument());
    await waitFor(
      () => expect(screen.getAllByText(/Подготовка файлов/).length).toBeGreaterThan(0),
      { timeout: 5000 },
    );
    expect(screen.getByText('Запуск…')).toBeInTheDocument();
  });

  it('hides install instructions when launcher is already linked', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(screen.getByText(/QXLauncher связан \(dev-99\)/)).toBeInTheDocument(),
    );
    expect(screen.queryByRole('button', { name: /Скачать QXLauncher/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/SignPath Foundation/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Установите QXLauncher/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Политика конфиденциальности/)).not.toBeInTheDocument();
  });

  it('shows player section, sorted instances, and link prompt when device is not linked', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const instances = [
      {
        id: 'inst-z',
        name: 'Zeta',
        mc_version: '1.21',
        loader: 'vanilla',
        created_at: 't',
        updated_at: 't',
      },
      {
        id: 'inst-a',
        name: 'Alpha',
        mc_version: '1.21',
        loader: 'fabric',
        loader_version: '0.16.0',
        created_at: 't',
        updated_at: 't',
      },
    ];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: instances }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Игрок')).toBeInTheDocument());
    const instancesPanel = document.querySelector('.launcher-panel--instances');
    expect(instancesPanel).toBeTruthy();
    await waitFor(() =>
      expect(within(instancesPanel as HTMLElement).getByText(/по алфавиту/)).toBeInTheDocument(),
    );
    expect(screen.getByText(/Свяжите QXLauncher на этом ПК/)).toBeInTheDocument();
    await waitFor(() => {
      const names = screen.getAllByText(/^(Alpha|Zeta)$/);
      expect(names[0]).toHaveTextContent('Alpha');
      expect(names[1]).toHaveTextContent('Zeta');
    });
    expect(screen.getByRole('link', { name: /Ресурсы/ })).toBeInTheDocument();
  });

  it('shows licensed launch hint when Microsoft account is not linked', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
      mockLauncherFetch((url) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/users/me/mojang')) {
          return new Response(JSON.stringify({ linked: false }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await waitFor(() =>
      expect(screen.getByRole('link', { name: /Привязать Microsoft/ })).toHaveAttribute('href', '/profile'),
    );
    expect(screen.getByText(/Привяжите Microsoft в профиле/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Играть/ })).toBeDisabled();
    expect(screen.queryByText('Играть как')).not.toBeInTheDocument();
    expect(screen.queryByText('Лицензия (Microsoft)')).not.toBeInTheDocument();
    expect(screen.queryByText('Оффлайн')).not.toBeInTheDocument();
  });

  it('shows launcher instances as cards by default and can switch to a list', async () => {
    const user = userEvent.setup({ delay: null, pointerEventsCheck: 0 });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(
            JSON.stringify({
              linked: true,
              device_id: 'dev-99',
              status: 'linked',
              launcher_version: '0.2.0',
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.getByText('u@test.com')).toBeInTheDocument();
    expect(screen.getByText(/QXLauncher связан \(dev-99\)/)).toBeInTheDocument();
    expect(screen.queryByText(/Инстансы синхронизируются/)).not.toBeInTheDocument();
    expect(document.querySelector('.launcher-hero-device')).toBeInTheDocument();
    expect(document.querySelector('.launcher-section--status')).not.toBeInTheDocument();
    expect(document.querySelector('.launcher-instance-list--cards')).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Карточки', checked: true })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Обновить$/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole('radio', { name: 'Список' }));

    expect(document.querySelector('.launcher-instance-list--cards')).not.toBeInTheDocument();
    expect(document.querySelector('.launcher-instance-list')).toBeInTheDocument();
    expect(window.localStorage.getItem(LAUNCHER_INSTANCES_VIEW_STORAGE_KEY)).toBe('list');
  });

  it('handles mojang status load failure in licensed mode', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/mojang')) {
          return new Response('fail', { status: 500 });
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Игрок')).toBeInTheDocument());
    await waitFor(() =>
      expect(screen.getByRole('link', { name: /Привязать Microsoft/ })).toBeInTheDocument(),
    );
  });

  it('shows linked Microsoft account in licensed player section', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/mojang')) {
          return new Response(
            JSON.stringify({
              linked: true,
              username: 'Notch',
              minecraft_uuid: 'uuid-notchy',
            }),
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Игрок')).toBeInTheDocument());
    await waitFor(() => expect(screen.getAllByText('Notch').length).toBeGreaterThan(0));
    expect(screen.getByText('Официальный аккаунт')).toBeInTheDocument();
    const playingAs = document.querySelector('.launcher-launch-bar');
    expect(playingAs?.querySelector('img')).toBeTruthy();
    expect(screen.queryByText('uuid-notchy')).not.toBeInTheDocument();
    expect(screen.queryByText(/Привяжите Microsoft в профиле/)).not.toBeInTheDocument();
  });

  it('launches with linked licensed mojang account', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
        if (url.includes('/users/me/mojang')) {
          return new Response(
            JSON.stringify({ linked: true, username: 'Notch', minecraft_uuid: 'uuid' }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return offlineProfileResponse();
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          const body = JSON.parse(String(init.body));
          expect(body.use_mojang_account).toBe(true);
          expect(body.offline_profile_id).toBeUndefined();
          return new Response(
            JSON.stringify({
              id: 'lr-lic',
              status: 'queued',
              instance_id: instance.id,
              expires_at: new Date().toISOString(),
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-lic')) {
          return new Response(
            JSON.stringify({
              id: 'lr-lic',
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await waitFor(() => expect(screen.getAllByText('Notch').length).toBeGreaterThan(0));
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() =>
      expect(vi.mocked(fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/launcher/launch-requests/lr-lic'),
        expect.anything(),
      ),
    );
  });

  it('defaults to linked Mojang account when offline profiles exist', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
        if (url.includes('/users/me/mojang')) {
          return new Response(
            JSON.stringify({ linked: true, username: 'Notch', minecraft_uuid: 'uuid' }),
            { status: 200 },
          );
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 'prof-1',
                  username: 'Steve',
                  model: 'steve',
                  created_at: 't',
                },
              ],
            }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return offlineProfileResponse();
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          const body = JSON.parse(String(init.body));
          expect(body.use_mojang_account).toBe(true);
          expect(body.offline_profile_id).toBeUndefined();
          return new Response(
            JSON.stringify({
              id: 'lr-lic',
              status: 'queued',
              instance_id: instance.id,
              expires_at: new Date().toISOString(),
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-lic')) {
          return new Response(
            JSON.stringify({
              id: 'lr-lic',
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await waitFor(() => expect(screen.getAllByText('Notch').length).toBeGreaterThan(0));
    expect(screen.getByText('Steve')).toBeInTheDocument();
    expect(document.querySelector('.launcher-profile-chip--selected .launcher-profile-name')).toHaveTextContent(
      /Notch/,
    );
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(fetch).toHaveBeenCalled());
  });

  it('launches offline profile after switching from linked Mojang default', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
        if (url.includes('/users/me/mojang')) {
          return new Response(
            JSON.stringify({ linked: true, username: 'Notch', minecraft_uuid: 'uuid' }),
            { status: 200 },
          );
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 'prof-1',
                  username: 'Steve',
                  model: 'steve',
                  offline_uuid: 'u1',
                  created_at: 't',
                },
              ],
            }),
            { status: 200 },
          );
        }
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return offlineProfileResponse();
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          const body = JSON.parse(String(init.body));
          expect(body.use_mojang_account).toBe(false);
          expect(body.offline_profile_id).toBe('prof-1');
          return new Response(
            JSON.stringify({
              id: 'lr-off',
              status: 'queued',
              instance_id: instance.id,
              expires_at: new Date().toISOString(),
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-off')) {
          return new Response(
            JSON.stringify({
              id: 'lr-off',
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await waitFor(() => expect(screen.getAllByText('Notch').length).toBeGreaterThan(0));
    await user.click(screen.getByText('Steve'));
    await waitFor(() => expect(screen.getAllByText('Steve').length).toBeGreaterThan(0));
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(fetch).toHaveBeenCalled());
  });

  it('unlinks launcher device from alert', async () => {
    const user = userEvent.setup({ delay: null });
  const successSpy = testMessage.success;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
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
  });

  it('shows unlink error message', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
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
  });

  it('shows generic unlink error for non-Error rejection', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    const { api } = await import('@/api/client');
    const unlinkSpy = vi.spyOn(api, 'unlinkDevice').mockRejectedValue('raw');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
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
    unlinkSpy.mockRestore();
  });

  it('defaults linked device status when omitted', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
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
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(
        screen.getByText('Нет профилей — добавьте свой ник'),
      ).toBeInTheDocument(),
    );
  });

  it('opens launcher download URL when configured', async () => {
    const openSpy = vi.spyOn(launcherDownload, 'openLauncherDownload').mockImplementation(() => {});
    vi.spyOn(launcherDownload, 'resolveLauncherDownloadUrl').mockReturnValue(
      'https://releases.example/qx-launcher.exe',
    );
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    const user = userEvent.setup({ delay: null });
    await waitFor(() => expect(screen.getByRole('button', { name: /Скачать QXLauncher/ })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(openSpy).toHaveBeenCalledWith('https://releases.example/qx-launcher.exe');
    openSpy.mockRestore();
    vi.restoreAllMocks();
  });

  it('shows launcher update banner when linked device is outdated', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/users/me/launcher-device')) {
          return new Response(
            JSON.stringify({
              linked: true,
              device_id: 'dev-update',
              status: 'linked',
              launcher_version: '0.1.0',
            }),
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Обновить/ })).toBeInTheDocument(),
    );
  });

  it('shows error when profiles fail to load', async () => {
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith('Не удалось загрузить профили'),
    );
  });

  it('shows generic play error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          return offlineProfileResponse();
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Backend unavailable'));
  });

  it('shows api error message when launch request fails', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          return offlineProfileResponse();
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({ error: { code: 'X', message: 'launcher offline' } }),
            { status: 400 },
          );
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('launcher offline'));
  });

  it('shows api error message when profile create fails', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
        if (url.includes('/launcher/profiles') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({ error: { code: 'X', message: 'username taken' } }),
            { status: 409 },
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await openAddProfileModal(user);
    await user.type(screen.getByLabelText('Никнейм'), 'Alex');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('username taken'));
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
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await openCreateInstanceModal(user);
    await user.type(screen.getByLabelText('Название'), 'Survival');
    await user.click(screen.getByRole('button', { name: 'Создать инстанс' }));
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.queryByRole('dialog', { name: 'Новый профиль игрока' })).not.toBeInTheDocument();
  });

  it('reports failed launch and deletes profile', async () => {
    const user = userEvent.setup({ delay: null });
  const successSpy = testMessage.success;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await openAddProfileModal(user);
    await user.type(screen.getByLabelText('Никнейм'), 'Steve');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await expectLauncherProfileListed('Steve');

    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(screen.getAllByText('JAVA_MISSING').length).toBeGreaterThan(0));
    expect(screen.getByText('Ошибка запуска')).toBeInTheDocument();

    const deleteButtons = screen.getAllByRole('button', { name: 'Удалить профиль?' });
    await user.click(deleteButtons[0]!);
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Профиль удалён'));
  });

  it('creates profile and launches instance', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await openAddProfileModal(user);
    await user.type(screen.getByLabelText('Никнейм'), 'Steve');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await expectLauncherProfileListed('Steve');

    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() =>
      expect(vi.mocked(fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/launcher/launch-requests/lr-1'),
        expect.anything(),
      ),
    );
  });

  it('handles expired launch request without toast', async () => {
    const user = userEvent.setup({ delay: null });
  const warningSpy = testMessage.warning;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          return offlineProfileResponse();
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() =>
      expect(vi.mocked(fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/launcher/launch-requests/lr-exp'),
        expect.anything(),
      ),
    );
    expect(warningSpy).not.toHaveBeenCalled();
  });

  it('shows launch progress in the instance card while polling', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
        if (url.includes('/launcher/profiles')) {
          return offlineProfileResponse();
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
        if (url.includes('/launcher/launch-requests/lr-run') && init?.method !== 'POST') {
          poll += 1;
          const status =
            poll === 1
              ? 'queued'
              : poll === 2
                ? 'dispatched'
                : poll === 3
                  ? 'preparing'
                  : poll === 4
                    ? 'downloading'
                    : poll === 5
                      ? 'launching'
                      : poll === 6
                        ? 'running'
                        : poll < 10
                          ? 'running'
                          : 'completed';
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(screen.getByText('Идёт запуск')).toBeInTheDocument());
    await waitFor(
      () => expect(screen.getAllByText('Minecraft запущен').length).toBeGreaterThan(0),
      { timeout: 12000 },
    );
  });

  it('shows raw message for unknown launch status', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const user = userEvent.setup({ delay: null });
    const instance = {
      id: 'inst-unknown',
      name: 'Unknown',
      loader: 'vanilla',
      mc_version: '1.21',
      created_at: 'now',
    };
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 'now',
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
        if (url.includes('/launcher/profiles')) {
          return offlineProfileResponse();
        }
        if (url.includes('/launcher/launch-requests') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 'lr-unknown',
              status: 'queued',
              instance_id: instance.id,
              expires_at: '2099-01-01T00:00:00Z',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/launcher/launch-requests/lr-unknown') && init?.method !== 'POST') {
          poll += 1;
          const status = poll === 1 ? 'custom_xyz' : 'completed';
          return new Response(
            JSON.stringify({
              id: 'lr-unknown',
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Unknown')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(screen.getAllByText('custom_xyz').length).toBeGreaterThan(0), {
      timeout: 8000,
    });
    // pollLaunchRequest keeps running after the assertion; wait for the terminal poll cycle.
    await new Promise((resolve) => setTimeout(resolve, 1700));
  });

  it('ignores linked device load failures', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
  });

  it('shows generic profile errors for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    const createSpy = vi.spyOn(api, 'createProfile').mockRejectedValueOnce('profile boom');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await openAddProfileModal(user);
    await user.type(screen.getByLabelText('Никнейм'), 'Alex');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось создать профиль'));
    createSpy.mockRestore();
  });

  it('shows generic delete profile error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    const deleteSpy = vi.spyOn(api, 'deleteProfile').mockRejectedValueOnce('delete boom');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const profile = {
      id: 'prof-1',
      username: 'Steve',
      offline_uuid: 'uuid',
      created_at: 't',
    };
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url, init) => {
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await user.click(getProfileDeleteButton());
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Не удалось удалить профиль'));
    deleteSpy.mockRestore();
  });

  it('shows delete profile api error', async () => {
    const user = userEvent.setup({ delay: null });
  const errorSpy = testMessage.error;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await user.click(getProfileDeleteButton());
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('delete denied'));
  });

  it('clears selected profile after deleting it', async () => {
    const user = userEvent.setup({ delay: null });
  const successSpy = testMessage.success;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    await user.click(getProfileDeleteButton());
    await user.click(await screen.findByRole('button', { name: 'OK' }));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Профиль удалён'));
  });

  it('does not open profile modal after create when profiles are missing', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = testMessage.info;
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Пока нет инстансов')).toBeInTheDocument());
    await openCreateInstanceModal(user);
    await user.type(screen.getByLabelText('Название'), 'Survival');
    await user.click(screen.getByRole('button', { name: 'Создать инстанс' }));
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(infoSpy).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog', { name: 'Новый профиль игрока' })).not.toBeInTheDocument();
  });

  it('shows generic failed launch error without error code', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
          return offlineProfileResponse();
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
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Играть/ }));
    await waitFor(() => expect(screen.getAllByText('Ошибка запуска').length).toBeGreaterThan(0));
  });

  it('logs in successfully', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/auth/login') && init?.method === 'POST') {
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
      if (url.includes('/users/me')) {
        return Promise.resolve(meResponse());
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

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
      expect(screen.getByRole('button', { name: 'U@, Меню аккаунта' })).toBeInTheDocument();
    });
    await waitForNoDialog();
  });

  it('shows login error message', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/auth/login') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { code: 'AUTH', message: 'invalid credentials' } }), {
            status: 401,
            statusText: 'Unauthorized',
          }),
        );
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

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

    await expectAuthModalError('invalid credentials');
  });

  it('shows generic login error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/auth/login') && init?.method === 'POST') {
        return Promise.reject(new TypeError('Failed to fetch'));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

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

    await expectAuthModalError('Сервер недоступен. Не удаётся связаться с API.');
  });

  it('registers successfully', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/auth/register') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              access_token: 'a',
              refresh_token: 'r',
              token_type: 'Bearer',
              expires_in: 3600,
            }),
            { status: 201 },
          ),
        );
      }
      if (url.includes('/users/me')) {
        return Promise.resolve(
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
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

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
      expect(screen.getByRole('button', { name: 'NE, Меню аккаунта' })).toBeInTheDocument();
    });
    await waitForNoDialog();
  });

  it('shows register error message', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/auth/register') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { code: 'AUTH', message: 'boom' } }), {
            status: 400,
            statusText: 'Bad Request',
          }),
        );
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

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

    await expectAuthModalError('boom');
  });

  it('shows generic register error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/auth/register') && init?.method === 'POST') {
        return Promise.reject(new TypeError('Failed to fetch'));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

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

    await expectAuthModalError('Сервер недоступен. Не удаётся связаться с API.');
  });

  it('shows profile spinner and content', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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

  it('selects launcher profile chip', async () => {
    const user = userEvent.setup({ delay: null });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const instance = {
      id: 'inst-1',
      name: 'Survival',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: 't',
      updated_at: 't',
    };
    const profiles = [
      { id: 'prof-1', username: 'Steve', offline_uuid: 'u1', created_at: 't' },
      { id: 'prof-2', username: 'Alex', offline_uuid: 'u2', created_at: 't', model: 'alex' as const },
    ];
    vi.mocked(fetch).mockImplementation(
      mockLauncherFetch((url) => {
        if (url.includes('/instances')) {
          return new Response(JSON.stringify({ items: [instance] }), { status: 200 });
        }
        if (url.includes('/launcher/profiles')) {
          return new Response(JSON.stringify({ items: profiles }), { status: 200 });
        }
        return null;
      }),
    );

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/launcher/*" element={<LauncherPage />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    await expectLauncherProfileListed('Steve');
    const alexChip = screen.getByText('Alex').closest('button');
    expect(alexChip).toBeTruthy();
    await user.click(alexChip!);
    await waitFor(() => {
      const selected = document.querySelector('.launcher-profile-chip--selected .launcher-profile-name');
      expect(selected).toHaveTextContent('Alex');
    });
  });
});
