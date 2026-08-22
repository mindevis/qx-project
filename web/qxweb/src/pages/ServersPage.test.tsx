import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { Routes, Route } from 'react-router-dom';
import { saveTokens, clearTokens } from '@/api/client';
import { renderWithProviders, waitForNoDialog } from '@/test/test-utils';
import { ServersPage } from './ServersPage';
import { GAME_SERVERS_LIST_VIEW_STORAGE_KEY } from '@/lib/installedResourcesView';

function requestUrl(input: RequestInfo | URL): string {
  return typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.href
      : input.url;
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

const sampleServer = {
  id: 'srv-1',
  name: 'Survival',
  status: 'pending',
  server_type: 'vanilla',
  mc_version: '1.21',
  agent_deployed: false,
  agent_online: false,
  ssh: { host: '1.2.3.4', port: 22, username: 'root' },
  config: { jar_path: '/opt/qxsystem/server/server.jar' },
};

function renderServers(route: string) {
  return renderWithProviders(
    <Routes>
      <Route path="/servers/*" element={<ServersPage />} />
    </Routes>,
    route,
  );
}

function mockAuthedFetch(handler: (url: string, init?: RequestInit) => Response | Promise<Response> | null) {
  return (input: RequestInfo | URL, init?: RequestInit) => {
    const url = requestUrl(input);
    const custom = handler(url, init);
    if (custom) {
      return Promise.resolve(custom);
    }
    if (url.includes('/networks')) {
      return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    }
    if (url.includes('/ollama')) {
      return Promise.resolve(
        new Response(JSON.stringify({ status: 'not_installed', models: [] }), { status: 200 }),
      );
    }
    if (url.includes('/mysql')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({ status: 'not_installed', databases: [], users: [], privilege_catalog: [] }),
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
    if (url.includes('/users/me')) {
      return Promise.resolve(meResponse());
    }
    return Promise.resolve(new Response('{}', { status: 200 }));
  };
}

class MockWebSocket {
  static OPEN = 1;
  static instances: MockWebSocket[] = [];
  readyState = MockWebSocket.OPEN;
  close = vi.fn();
  constructor(_url: string) {
    MockWebSocket.instances.push(this);
  }
  send() {}
}

async function clickAddDedicated(user: ReturnType<typeof userEvent.setup>) {
  await waitFor(() => {
    expect(screen.getAllByRole('button', { name: /Добавить выделенный сервер/i }).length).toBeGreaterThan(0);
  });
  const buttons = screen.getAllByRole('button', { name: /Добавить выделенный сервер/i });
  await user.click(buttons[0]!);
}

describe('ServersPage', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    window.localStorage.removeItem(GAME_SERVERS_LIST_VIEW_STORAGE_KEY);
    vi.stubGlobal('fetch', vi.fn());
    vi.stubGlobal('WebSocket', MockWebSocket);
    clearTokens();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('prompts login when unauthenticated', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation(() => Promise.resolve(meResponse()));
    renderServers('/servers');
    expect(await screen.findByText('Нужен аккаунт')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Войти/i }));
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('lists servers when api omits items array', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers') && !url.includes('/servers/')) {
          return new Response(JSON.stringify({}), { status: 200 });
        }
        return null;
      }),
    );

    renderServers('/servers');
    await waitFor(() =>
      expect(
        screen.getByText('Добавьте Linux выделенный сервер с SSH-доступом — мы установим QXAgent и подготовим сервер к запуску.'),
      ).toBeInTheDocument(),
    );
  });

  it('creates server with default ssh port when omitted', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    let postedBody = '';
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          postedBody = init.body as string;
          return new Response(
            JSON.stringify({ ...sampleServer, id: 'srv-new', name: 'New dedicated server' }),
            { status: 201 },
          );
        }
        if (url.includes('/servers/srv-new')) {
          return new Response(
            JSON.stringify({ ...sampleServer, id: 'srv-new', name: 'New dedicated server' }),
            { status: 200 },
          );
        }
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');
    await waitFor(() => expect(screen.getByText('Ваши серверы')).toBeInTheDocument());
    await clickAddDedicated(user);
    await user.type(screen.getByLabelText('Название'), 'New dedicated server');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.clear(screen.getByLabelText('SSH Port'));
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(postedBody).toContain('"port":22'));
  });

  it('shows spinner while auth is loading', async () => {
    vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });

    renderServers('/servers');
    expect(document.querySelector('.ant-spin')).toBeTruthy();
  });

  it('lists servers and opens create modal', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && (!init?.method || init.method === 'GET') && !url.includes('/console')) {
          return new Response(JSON.stringify({ items: [sampleServer] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.getAllByText('Не развёрнут').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Новый').length).toBeGreaterThan(0);

    await clickAddDedicated(user);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Close' }));
    await waitForNoDialog();
  });

  it('opens dedicated host from the card and does not show a leftover loader label', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: true, agent_online: true }), {
            status: 200,
          });
        }
        if (url.includes('/servers') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              items: [{ ...sampleServer, server_type: 'forge', agent_deployed: true, agent_online: true }],
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');

    await waitFor(() => expect(screen.getByRole('link', { name: /Survival/ })).toBeInTheDocument());
    expect(screen.queryByText(/^forge$/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Открыть' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('link', { name: /Survival/ }));
    await waitFor(() => expect(screen.getByText('Игровые серверы')).toBeInTheDocument());
  });

  it('shows error when server list fails', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers')) {
          return new Response('fail', { status: 500, statusText: 'Server Error' });
        }
        return null;
      }),
    );

    renderServers('/servers');
    await waitFor(() =>
      expect(testMessage.error).toHaveBeenCalledWith('Не удалось загрузить серверы'),
    );
  });

  it('creates server and navigates to detail', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              id: 'srv-new',
              name: 'New dedicated server',
              agent_deployed: true,
              agent_online: true,
              status: 'online',
            }),
            { status: 201 },
          );
        }
        if (url.includes('/servers/srv-new')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              id: 'srv-new',
              name: 'New dedicated server',
              agent_deployed: true,
              agent_online: true,
              status: 'online',
            }),
            { status: 200 },
          );
        }
        if (url.includes('/servers') && (!init?.method || init.method === 'GET')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');

    await waitFor(() => expect(screen.getByText('Ваши серверы')).toBeInTheDocument());
    await clickAddDedicated(user);
    await user.type(screen.getByLabelText('Название'), 'New dedicated server');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(screen.getByText('New dedicated server')).toBeInTheDocument());
    expect(screen.getAllByText('Онлайн').length).toBeGreaterThan(0);
  });

  it('renders server detail actions when agent is online', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    let server = { ...sampleServer, agent_deployed: true, agent_online: true };
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/servers/srv-1/deploy')) {
          server = { ...server, status: 'offline' };
          return new Response(JSON.stringify(server), { status: 200 });
        }
        if (url.includes('/servers/srv-1/start')) {
          return new Response(JSON.stringify({ status: 'starting' }), { status: 200 });
        }
        if (url.includes('/servers/srv-1/stop')) {
          return new Response(JSON.stringify({ status: 'stopping' }), { status: 200 });
        }
        if (url.includes('/servers/srv-1/restart')) {
          return new Response(JSON.stringify({ status: 'starting' }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify(server), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    const view = renderServers('/servers/srv-1');

    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.getAllByText('Онлайн').length).toBeGreaterThan(0);
    expect(screen.queryByRole('button', { name: /Deploy agent/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Start/i })).not.toBeInTheDocument();
    expect(screen.getByText('Игровые серверы')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Добавить игровой сервер/i })).toBeInTheDocument();
    view.unmount();
  });

  it('adds a game server when agent is online', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    let gameServers = { items: [] as Record<string, unknown>[] };
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/launcher/mc-versions')) {
          return new Response(
            JSON.stringify({
              latest: { release: '1.21' },
              items: [
                { id: '1.21', type: 'release' },
                { id: '1.20.4', type: 'release' },
              ],
            }),
            { status: 200 },
          );
        }
        if (url.includes('/versions/') && url.includes('/builds')) {
          return new Response(JSON.stringify([{ id: 456, channel: 'STABLE' }]), { status: 200 });
        }
        if (
          url.includes('/upstream/papermc/v3/projects/paper') ||
          url.includes('/upstream/papermc/v2/projects/paper') ||
          url.includes('api.papermc.io/v2/projects/paper') ||
          url.includes('fill.papermc.io/v3/projects/paper')
        ) {
          return new Response(
            JSON.stringify({ versions: { '1.21': ['1.21'], '1.20': ['1.20.4'] } }),
            { status: 200 },
          );
        }
        if (url.includes('/servers/srv-1/game-servers') && init?.method === 'POST') {
          const created = {
            id: 'gs-1',
            name: 'Survival MC',
            server_type: 'paper',
            mc_version: '1.21',
            loader_version: '456',
            address: '1.2.3.4',
            port: 25565,
            status: 'installing',
            created_at: '2026-01-01T00:00:00Z',
          };
          gameServers = { items: [created] };
          return new Response(JSON.stringify(created), { status: 201 });
        }
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify(gameServers), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: true, agent_online: true }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Добавить игровой сервер/i }));
    await user.type(screen.getByLabelText('Название'), 'Survival MC');
    const dialog = screen.getByRole('dialog');
    const comboboxes = within(dialog).getAllByRole('combobox');
    await user.click(comboboxes[0]!);
    await user.click(await screen.findByText('Paper'));
    await waitFor(() => {
      expect(within(dialog).getByText('#456')).toBeInTheDocument();
    });
    await user.click(within(dialog).getByRole('button', { name: /Добавить игровой сервер/i }));

    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());
    expect(screen.getByText('Survival MC')).toBeInTheDocument();
    expect(screen.getByText('Paper')).toBeInTheDocument();
    expect(screen.getAllByText('1.21').length).toBeGreaterThan(0);
    expect(screen.getByText('456')).toBeInTheDocument();
    expect(screen.getByText((_, el) => el?.textContent === '1.2.3.4:25565')).toBeInTheDocument();
  });

  it('shows offline agent hint on detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: false, agent_online: false, status: 'pending' }), { status: 200 });
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getAllByText('Не развёрнут').length).toBeGreaterThan(0));
    expect(screen.getByText(/После Deploy агент подключится по WSS автоматически/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Deploy agent/i })).toBeInTheDocument();
  });

  it('shows deploy error when agent is offline', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/deploy')) {
          return new Response(
            JSON.stringify({
              error: {
                code: 'HOST_NOT_LINUX',
                message: 'QX agent requires a Linux dedicated server',
              },
            }),
            { status: 422 },
          );
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: false, agent_online: false, status: 'pending' }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Deploy agent/i }));
    await waitFor(() =>
      expect(testMessage.error).toHaveBeenCalledWith('QX agent requires a Linux dedicated server'),
    );
  });

  it('redirects when server missing', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/missing')) {
          return new Response('not found', { status: 404, statusText: 'Not Found' });
        }
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderServers('/servers/missing');
    await waitFor(() =>
      expect(testMessage.error).toHaveBeenCalledWith('Сервер не найден'),
    );
  });

  it('covers status tag variants', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    const statuses = ['online', 'starting', 'deploying', 'stopping', 'error', 'pending', 'custom'];
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers')) {
          return new Response(
            JSON.stringify({
              items: statuses.map((status, i) => ({
                ...sampleServer,
                id: `srv-${i}`,
                name: `Server ${status}`,
                status,
                agent_deployed: status !== 'pending',
                agent_online: status === 'online',
              })),
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderServers('/servers');
    await waitFor(() => expect(screen.getByText('Server online')).toBeInTheDocument());
    expect(screen.getByText('Server custom')).toBeInTheDocument();
    expect(screen.getAllByText('Оффлайн').length).toBeGreaterThan(0);
    expect(screen.getByText('Развёртывается…')).toBeInTheDocument();
  });

  it('shows dedicated host offline on detail for unknown backend status without agent', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({ ...sampleServer, status: 'custom_detail', agent_online: false }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getAllByText('Не развёрнут').length).toBeGreaterThanOrEqual(1));
  });

  it('deletes server from detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1') && init?.method === 'DELETE') {
          return new Response(null, { status: 204 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: true, agent_online: true, status: 'online' }), {
            status: 200,
          });
        }
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    const view = renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Удалить/i }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));

    await waitFor(() => expect(testMessage.success).toHaveBeenCalledWith('Сервер удалён'));
    view.unmount();
  });

  it('shows create validation and api errors', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({ error: { code: 'BAD', message: 'invalid ssh key' } }),
            { status: 400 },
          );
        }
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');
    await waitFor(() => expect(screen.getByText('Ваши серверы')).toBeInTheDocument());

    await clickAddDedicated(user);
    await user.type(screen.getByLabelText('Название'), 'Bad dedicated server');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('invalid ssh key'));
  });

  it('shows action and delete errors on detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1') && init?.method === 'DELETE') {
          return new Response(
            JSON.stringify({ error: { code: 'FAIL', message: 'delete failed' } }),
            { status: 500 },
          );
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_deployed: true,
              agent_online: true,
              minecraft_running: true,
              config: {},
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: /Deploy agent/i })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Удалить/i }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('delete failed'));
  });

  it('shows generic action and delete errors for non-error throws', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1') && init?.method === 'DELETE') {
          return Promise.reject('delete boom');
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_deployed: true,
              agent_online: true,
              minecraft_running: true,
              config: {},
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Удалить/i }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('Backend unavailable'));
  });

  it('ignores create errors without message', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          return new Response(JSON.stringify({ error: { code: 'X', message: '' } }), {
            status: 500,
            statusText: 'Error',
          });
        }
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');
    await waitFor(() => expect(screen.getByText('Ваши серверы')).toBeInTheDocument());

    await clickAddDedicated(user);
    await user.type(screen.getByLabelText('Название'), 'Silent fail');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(testMessage.error).not.toHaveBeenCalled());
  });

  it('hides game server console until instance is running', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 'gs-1',
                  name: 'qRPG',
                  server_type: 'forge',
                  mc_version: '1.20.1',
                  loader_version: '47.4.20',
                  address: 'localhost',
                  port: 25565,
                  status: 'stopped',
                  created_at: 'now',
                },
              ],
            }),
            { status: 200 },
          );
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_deployed: true,
              agent_online: true,
              status: 'online',
              minecraft_running: true,
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('qRPG')).toBeInTheDocument());
    expect(screen.queryByPlaceholderText('Команда сервера (Enter)')).not.toBeInTheDocument();
  });

  it('shows game server console inside running instance card', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 'gs-1',
                  name: 'qRPG',
                  server_type: 'forge',
                  mc_version: '1.20.1',
                  loader_version: '47.4.20',
                  address: 'localhost',
                  port: 25565,
                  status: 'running',
                  created_at: 'now',
                },
              ],
            }),
            { status: 200 },
          );
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_deployed: true,
              agent_online: true,
              status: 'online',
              minecraft_running: true,
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1/game-servers/gs-1');
    await waitFor(() =>
      expect(screen.getByPlaceholderText('Команда сервера (Enter)')).toBeInTheDocument(),
    );
    expect(screen.getByRole('tab', { name: /RCON консоль/i })).toBeInTheDocument();
  });

  it('polls server detail on interval', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    let detailCalls = 0;
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers') || url.includes('/networks')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
    if (url.includes('/ollama')) {
      return null;
    }
    if (url.includes('/mysql')) {
      return null;
    }
        if (url.includes('/servers/srv-1')) {
          detailCalls += 1;
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: true, agent_online: true }), { status: 200 });
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    expect(detailCalls).toBe(1);

    await vi.advanceTimersByTimeAsync(5000);
    await waitFor(() => expect(detailCalls).toBeGreaterThan(1));
    vi.useRealTimers();
  });

  it('redirects unknown nested routes to list', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    renderServers('/servers/extra/nested');
    await waitFor(() => expect(screen.getByText('Ваши серверы')).toBeInTheDocument());
  });

  it('updates deployed agent from detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1/deploy') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_deployed: true,
              agent_online: true,
              status: 'online',
            }),
            { status: 200 },
          );
        }
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_deployed: true,
              agent_online: true,
              status: 'online',
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Обновить QXAgent/i }));
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());
  });

  const onlineVps = {
    ...sampleServer,
    agent_deployed: true,
    agent_online: true,
    status: 'online',
    created_at: '2026-01-01T00:00:00Z',
    last_seen_at: '2026-01-02T00:00:00Z',
  };

  const stoppedGame = {
    id: 'gs-1',
    name: 'Stopped MC',
    server_type: 'paper',
    mc_version: '1.21',
    loader_version: '456',
    address: '1.2.3.4',
    port: 25565,
    status: 'stopped',
    created_at: '2026-01-01T00:00:00Z',
  };

  function mockOnlineDetail(
    games: Record<string, unknown>[] = [stoppedGame],
    handler?: (url: string, init?: RequestInit) => Response | null,
  ) {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        const custom = handler?.(url, init);
        if (custom) return custom;
        if (url.includes('/launcher/mc-versions')) {
          return new Response(
            JSON.stringify({
              latest: { release: '1.21' },
              items: [{ id: '1.21', type: 'release' }],
            }),
            { status: 200 },
          );
        }
        if (url.includes('/servers/srv-1/game-servers') && !url.includes('/gs-1')) {
          return new Response(JSON.stringify({ items: games }), { status: 200 });
        }
    if (url.includes('/ollama')) {
      return null;
    }
    if (url.includes('/mysql')) {
      return null;
    }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify(onlineVps), { status: 200 });
        }
        return null;
      }),
    );
  }

  it('shows empty and loading game server states', async () => {
    mockOnlineDetail([]);
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Пока нет игровых серверов')).toBeInTheDocument());

    let resolveGames: (value: Response) => void = () => undefined;
    const gamesPromise = new Promise<Response>((resolve) => {
      resolveGames = resolve;
    });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/game-servers')) {
          return gamesPromise;
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify(onlineVps), { status: 200 });
        }
        return null;
      }),
    );

    const view = renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    resolveGames(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    await waitFor(() => expect(screen.getByText('Пока нет игровых серверов')).toBeInTheDocument());
    view.unmount();
  });

  it('creates vanilla game server without loader version', async () => {
    let gameServers = { items: [] as Record<string, unknown>[] };
    mockOnlineDetail([], (url, init) => {
      if (url.includes('/servers/srv-1/game-servers') && init?.method === 'POST') {
        const created = {
          id: 'gs-vanilla',
          name: 'Vanilla World',
          server_type: 'vanilla',
          mc_version: '1.21',
          address: '1.2.3.4',
          port: 25566,
          status: 'installing',
          created_at: '2026-01-01T00:00:00Z',
        };
        gameServers = { items: [created] };
        return new Response(JSON.stringify(created), { status: 201 });
      }
      if (url.includes('/servers/srv-1/game-servers')) {
        return new Response(JSON.stringify(gameServers), { status: 200 });
      }
      return null;
    });

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Добавить игровой сервер/i }));
    await user.type(screen.getByLabelText('Название'), 'Vanilla World');
    const dialog = screen.getByRole('dialog');
    const comboboxes = within(dialog).getAllByRole('combobox');
    await user.click(comboboxes[0]!);
    const vanillaOptions = await screen.findAllByText('Vanilla');
    await user.click(vanillaOptions[vanillaOptions.length - 1]!);
    await user.click(within(dialog).getByRole('button', { name: /Добавить игровой сервер/i }));
    await waitFor(() => expect(screen.getByText('Vanilla World')).toBeInTheDocument());
  });

  it('shows game servers as cards by default and switches to a table list', async () => {
    mockOnlineDetail();
    const user = userEvent.setup({ delay: null, pointerEventsCheck: 0 });
    renderServers('/servers/srv-1');

    await waitFor(() => expect(screen.getByText('Stopped MC')).toBeInTheDocument());
    expect(document.querySelector('.servers-game-list--cards')).toBeInTheDocument();
    expect(document.querySelector('.servers-game-table')).not.toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Карточки', checked: true })).toBeInTheDocument();

    await user.click(screen.getByRole('radio', { name: 'Список' }));

    expect(document.querySelector('.servers-game-table')).toBeInTheDocument();
    expect(document.querySelector('.servers-game-list--cards')).not.toBeInTheDocument();
    expect(window.localStorage.getItem(GAME_SERVERS_LIST_VIEW_STORAGE_KEY)).toBe('list');
  });

  it('installs ollama from the dedicated server page', async () => {
    let ollama = { status: 'not_installed', models: [] as unknown[] };
    mockOnlineDetail([], (url, init) => {
      if (url.includes('/ollama/install') && init?.method === 'POST') {
        ollama = { status: 'installing', models: [] };
        return new Response(JSON.stringify(ollama), { status: 202 });
      }
      if (url.includes('/ollama')) {
        return new Response(JSON.stringify(ollama), { status: 200 });
      }
      return null;
    });

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Ollama')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Установить Ollama/i }));
    await waitFor(() => expect(testMessage.success).toHaveBeenCalledWith('Ollama установлена и запускается'));
  });

  it('installs mysql from the dedicated server page', async () => {
    let mysql = { status: 'not_installed', databases: [] as unknown[], users: [] as unknown[], privilege_catalog: [] as string[] };
    mockOnlineDetail([], (url, init) => {
      if (url.includes('/mysql/install') && init?.method === 'POST') {
        mysql = { status: 'installing', databases: [], users: [], privilege_catalog: [] };
        return new Response(JSON.stringify(mysql), { status: 202 });
      }
      if (url.includes('/mysql')) {
        return new Response(JSON.stringify(mysql), { status: 200 });
      }
      return null;
    });

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('MySQL')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Установить MySQL/i }));
    await waitFor(() => expect(testMessage.success).toHaveBeenCalledWith('MySQL установлена и запускается'));
  });

  it('clones game server from the card list', async () => {
    mockOnlineDetail([stoppedGame], (url, init) => {
      if (url.includes('/game-servers/gs-1/clone') && init?.method === 'POST') {
        return new Response(
          JSON.stringify({ ...stoppedGame, id: 'gs-2', name: 'Stopped MC (copy)', port: 25566 }),
          { status: 201 },
        );
      }
      return null;
    });

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Stopped MC')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Клонировать' }));
    await user.click((await screen.findAllByRole('button', { name: /^OK$/i })).at(-1)!);
    await waitFor(() => expect(testMessage.success).toHaveBeenCalledWith('Игровой сервер скопирован'));
  });

  it('refreshes servers list from workspace', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers') && !url.includes('/servers/')) {
          return new Response(JSON.stringify({ items: [sampleServer] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getAllByRole('button', { name: /Обновить/i })[0]!);
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());
  });

  it('deploys agent from workflow panel', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1/deploy') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({ ...sampleServer, agent_deployed: true, agent_online: true }),
            { status: 200 },
          );
        }
        if (url.includes('/servers/srv-1/game-servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_deployed: false, agent_online: false }), {
            status: 200,
          });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Deploy agent/i }));
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());
  });
});
