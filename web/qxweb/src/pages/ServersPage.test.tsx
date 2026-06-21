import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { Routes, Route } from 'react-router-dom';
import { saveTokens, clearTokens } from '@/api/client';
import { renderWithProviders, waitForNoDialog } from '@/test/test-utils';
import { ServersPage } from './ServersPage';

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
  status: 'offline',
  server_type: 'vanilla',
  mc_version: '1.21',
  agent_online: false,
  ssh: { host: '1.2.3.4', port: 22, username: 'root' },
  config: { jar_path: '/opt/qx/server/server.jar' },
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

describe('ServersPage', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal('fetch', vi.fn());
    vi.stubGlobal('WebSocket', MockWebSocket);
    clearTokens();
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('prompts login for guests', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation(() => Promise.resolve(meResponse()));
    renderServers('/servers');
    expect(await screen.findByText('Управление серверами доступно после входа.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Войти' }));
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('lists servers when api omits items array', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
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
    await waitFor(() => expect(screen.getByText('Нет серверов — добавьте Linux VPS с SSH-доступом')).toBeInTheDocument());
  });

  it('creates server with default ssh port when omitted', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    let postedBody = '';
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          postedBody = init.body as string;
          return new Response(
            JSON.stringify({ ...sampleServer, id: 'srv-new', name: 'New VPS' }),
            { status: 201 },
          );
        }
        if (url.includes('/servers/srv-new')) {
          return new Response(
            JSON.stringify({ ...sampleServer, id: 'srv-new', name: 'New VPS' }),
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
    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());
    await user.click(screen.getByText('Добавить VPS'));
    await user.type(screen.getByLabelText('Название'), 'New VPS');
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
      expires_in: 60,
    });

    renderServers('/servers');
    expect(document.querySelector('.ant-spin')).toBeTruthy();
  });

  it('lists servers and opens create modal', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
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
    expect(screen.getByText('Оффлайн')).toBeInTheDocument();

    await user.click(screen.getByText('Добавить VPS'));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Close' }));
    await waitForNoDialog();
  });

  it('shows error when server list fails', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
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
      expect(message.error).toHaveBeenCalledWith('Не удалось загрузить серверы'),
    );
  });

  it('creates server and navigates to detail', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({ ...sampleServer, id: 'srv-new', name: 'New VPS', agent_online: true, status: 'online' }),
            { status: 201 },
          );
        }
        if (url.includes('/servers/srv-new')) {
          return new Response(
            JSON.stringify({ ...sampleServer, id: 'srv-new', name: 'New VPS', agent_online: true, status: 'online' }),
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

    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());
    await user.click(screen.getByText('Добавить VPS'));
    await user.type(screen.getByLabelText('Название'), 'New VPS');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(screen.getByText('New VPS')).toBeInTheDocument());
    expect(screen.getByText('Agent подключён')).toBeInTheDocument();
  });

  it('renders server detail actions when agent is online', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    let server = { ...sampleServer, agent_online: true };
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
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
    expect(screen.getByText('Agent подключён')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Deploy agent/i }));
    await waitFor(() => expect(message.success).toHaveBeenCalled());
    expect(screen.queryByRole('button', { name: 'Start' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Stop' })).not.toBeInTheDocument();
    view.unmount();
  });

  it('shows offline agent hint on detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_online: false }), { status: 200 });
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Agent оффлайн')).toBeInTheDocument());
    expect(screen.getByText(/После Deploy агент подключится по WSS автоматически/)).toBeInTheDocument();
  });

  it('stops server from detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/stop')) {
          return new Response(JSON.stringify({ status: 'stopping' }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
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

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByText('Stop'));
    await waitFor(() => expect(message.success).toHaveBeenCalled());
  });

  it('restarts server from detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1/restart')) {
          return new Response(JSON.stringify({ status: 'starting' }), { status: 200 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
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

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());
    await user.click(screen.getByText('Restart'));
    await waitFor(() => expect(message.success).toHaveBeenCalled());
  });

  it('redirects when server missing', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
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
      expect(message.error).toHaveBeenCalledWith('Сервер не найден'),
    );
  });

  it('covers status tag variants', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
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
    expect(screen.getByText('Deploy…')).toBeInTheDocument();
  });

  it('deletes server from detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1') && init?.method === 'DELETE') {
          return new Response(null, { status: 204 });
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(JSON.stringify({ ...sampleServer, agent_online: true, status: 'online' }), {
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

    await waitFor(() => expect(message.success).toHaveBeenCalledWith('Сервер удалён'));
    view.unmount();
  });

  it('shows create validation and api errors', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
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
    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());

    await user.click(screen.getByText('Добавить VPS'));
    await user.type(screen.getByLabelText('Название'), 'Bad VPS');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.type(screen.getByLabelText('JVM args (по одному на строку)'), '-Xmx2G\n-Xms1G');
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(message.error).toHaveBeenCalledWith('invalid ssh key'));
  });

  it('shows action and delete errors on detail page', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1/deploy')) {
          return new Response(
            JSON.stringify({
              error: {
                code: 'HOST_NOT_LINUX',
                message: 'QX agent requires a Linux VPS',
              },
            }),
            { status: 422 },
          );
        }
        if (url.includes('/servers/srv-1/stop')) {
          return new Response(
            JSON.stringify({ error: { code: 'FAIL', message: 'stop failed' } }),
            { status: 500 },
          );
        }
        if (url.includes('/servers/srv-1') && init?.method === 'DELETE') {
          return new Response(
            JSON.stringify({ error: { code: 'FAIL', message: 'delete failed' } }),
            { status: 500 },
          );
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({ ...sampleServer, agent_online: true, minecraft_running: true, config: {} }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /Deploy agent/i }));
    await waitFor(() =>
      expect(message.error).toHaveBeenCalledWith('QX agent requires a Linux VPS'),
    );

    await user.click(screen.getByRole('button', { name: /^Stop$/i }));
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('stop failed'));

    await user.click(screen.getByRole('button', { name: /Удалить/i }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('delete failed'));
  });

  it('shows generic action and delete errors for non-error throws', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers/srv-1/stop')) {
          return Promise.reject('stop boom');
        }
        if (url.includes('/servers/srv-1') && init?.method === 'DELETE') {
          return Promise.reject('delete boom');
        }
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({ ...sampleServer, agent_online: true, minecraft_running: true, config: {} }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Survival')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: /^Stop$/i }));
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('Ошибка'));

    await user.click(screen.getByRole('button', { name: /Удалить/i }));
    await user.click(screen.getByRole('button', { name: /^OK$/i }));
    await waitFor(() => expect(message.error).toHaveBeenCalledTimes(2));
  });

  it('ignores create errors without message', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url, init) => {
        if (url.includes('/servers') && init?.method === 'POST') {
          return Promise.reject(new Error(''));
        }
        if (url.includes('/servers')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return null;
      }),
    );

    const user = userEvent.setup({ delay: null });
    renderServers('/servers');
    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());

    await user.click(screen.getByText('Добавить VPS'));
    await user.type(screen.getByLabelText('Название'), 'Silent fail');
    await user.type(screen.getByLabelText('SSH Host'), '10.0.0.1');
    await user.type(screen.getByLabelText('SSH User'), 'ubuntu');
    await user.type(
      screen.getByLabelText('SSH Private Key'),
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----',
    );
    await user.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(message.error).not.toHaveBeenCalled());
  });

  it('shows server console when minecraft is starting', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1')) {
          return new Response(
            JSON.stringify({
              ...sampleServer,
              agent_online: true,
              status: 'starting',
              minecraft_running: false,
            }),
            { status: 200 },
          );
        }
        return null;
      }),
    );

    renderServers('/servers/srv-1');
    await waitFor(() => expect(screen.getByText('Консоль')).toBeInTheDocument());
  });

  it('polls server detail on interval', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    let detailCalls = 0;
    vi.mocked(fetch).mockImplementation(
      mockAuthedFetch((url) => {
        if (url.includes('/servers/srv-1')) {
          detailCalls += 1;
          return new Response(JSON.stringify({ ...sampleServer, agent_online: true }), { status: 200 });
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
      expires_in: 60,
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
    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());
  });
});
