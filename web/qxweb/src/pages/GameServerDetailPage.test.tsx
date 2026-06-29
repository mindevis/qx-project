import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { Routes, Route } from 'react-router-dom';
import { saveTokens, clearTokens } from '@/api/client';
import { renderWithProviders } from '@/test/test-utils';
import { GameServerDetailPage } from './GameServerDetailPage';

const vpsServer = {
  id: 'srv-1',
  name: 'VPS',
  status: 'online',
  server_type: 'vanilla',
  mc_version: '1.21',
  agent_deployed: true,
  agent_online: true,
  ssh: { host: '1.2.3.4', port: 22, username: 'root' },
  config: {},
};

const gameServer = {
  id: 'gs-1',
  name: 'Forge RPG',
  server_type: 'forge',
  mc_version: '1.20.1',
  loader_version: '47.2.0',
  address: 'play.example.com',
  port: 25565,
  rcon_password: 'secret',
  rcon_port: 25575,
  status: 'stopped',
  created_at: 'now',
};

function requestUrl(input: RequestInfo | URL): string {
  return typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.href
      : input.url;
}

function renderDetail(route = '/servers/srv-1/game-servers/gs-1') {
  return renderWithProviders(
    <Routes>
      <Route path="/servers/:id/game-servers/:gameServerId" element={<GameServerDetailPage />} />
      <Route path="/servers/:id" element={<div>VPS detail</div>} />
      <Route path="/servers" element={<div>Servers list</div>} />
    </Routes>,
    route,
  );
}

class MockWebSocket {
  static OPEN = 1;
  static instances: MockWebSocket[] = [];
  readyState = MockWebSocket.OPEN;
  close = vi.fn();
  onmessage: ((ev: { data: string }) => void) | null = null;
  constructor(_url: string) {
    MockWebSocket.instances.push(this);
    queueMicrotask(() => {
      this.onmessage?.({ data: JSON.stringify({ type: 'status', status: 'connected' }) });
    });
  }
  send() {}
}

describe('GameServerDetailPage', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal('fetch', vi.fn());
    vi.stubGlobal('WebSocket', MockWebSocket);
    clearTokens();
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    clearTokens();
  });

  it('loads game server detail and runs power actions', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'u@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/servers/srv-1/game-servers') && !url.includes('/gs-1/')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [gameServer] }), { status: 200 }));
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/stop') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ ...gameServer, status: 'stopped' }), { status: 200 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/start') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ ...gameServer, status: 'starting' }), { status: 200 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/restart') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ ...gameServer, status: 'starting' }), { status: 200 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/reinstall') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ ...gameServer, status: 'installing' }), { status: 200 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1') && init?.method === 'DELETE') {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      if (url.includes('/servers/srv-1')) {
        return Promise.resolve(new Response(JSON.stringify(vpsServer), { status: 200 }));
      }
      if (url.includes('/properties')) {
        return Promise.resolve(new Response(JSON.stringify({ properties: [] }), { status: 200 }));
      }
      if (url.includes('/mods')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
      }
      if (url.includes('/files')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderDetail();
    await waitFor(() => expect(screen.getByText('Forge RPG')).toBeInTheDocument());
    expect(screen.getByText(/play\.example\.com:25565/)).toBeInTheDocument();
    expect(screen.getByText(/secret/)).toBeInTheDocument();

    await user.click(screen.getAllByRole('button', { name: /Запустить/i })[0]!);
    await waitFor(() => expect(message.success).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /Перезапустить/i }));
    await waitFor(() => expect(message.success).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /Удалить/ }));
    await user.click((await screen.findAllByRole('button', { name: /^OK$/i })).at(-1)!);
    await waitFor(() => expect(screen.getByText('VPS detail')).toBeInTheDocument());
  });

  it('redirects when game server is missing', async () => {
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'u@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/game-servers')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
      }
      if (url.includes('/servers/srv-1')) {
        return Promise.resolve(new Response(JSON.stringify(vpsServer), { status: 200 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderDetail();
    await waitFor(() => expect(screen.getByText('VPS detail')).toBeInTheDocument());
    expect(message.error).toHaveBeenCalled();
  });

  it('redirects on load failure', async () => {
    vi.mocked(fetch).mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'u@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/servers/srv-1')) {
        return Promise.resolve(new Response('fail', { status: 500 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderDetail();
    await waitFor(() => expect(screen.getByText('VPS detail')).toBeInTheDocument());
    expect(message.error).toHaveBeenCalled();
  });

  it('shows power action errors', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'u@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/game-servers') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { message: 'power failed' } }), { status: 500 }),
        );
      }
      if (url.includes('/game-servers')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [gameServer] }), { status: 200 }));
      }
      if (url.includes('/servers/srv-1')) {
        return Promise.resolve(new Response(JSON.stringify(vpsServer), { status: 200 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderDetail();
    await waitFor(() => expect(screen.getByText('Forge RPG')).toBeInTheDocument());
    await user.click(screen.getAllByRole('button', { name: /Запустить/i })[0]!);
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('power failed'));
  });

  it('reinstalls game server and stops running instance', async () => {
    const user = userEvent.setup({ delay: null });
    const runningGame = { ...gameServer, status: 'running' };
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'u@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/stop') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ ...runningGame, status: 'stopped' }), { status: 200 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/reinstall') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ ...runningGame, status: 'installing' }), { status: 200 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [runningGame] }), { status: 200 }));
      }
      if (url.includes('/servers/srv-1')) {
        return Promise.resolve(new Response(JSON.stringify(vpsServer), { status: 200 }));
      }
      if (url.includes('/properties')) {
        return Promise.resolve(new Response(JSON.stringify({ properties: [] }), { status: 200 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderDetail();
    await waitFor(() => expect(screen.getByText('Forge RPG')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Остановить/i }));
    await waitFor(() => expect(message.success).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /Переустановить/i }));
    await user.click((await screen.findAllByRole('button', { name: /^OK$/i })).at(-1)!);
    await waitFor(() => expect(message.success).toHaveBeenCalledTimes(2));
  });

  it('shows delete and reinstall errors', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockImplementation((input, init) => {
      const url = requestUrl(input);
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'u@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1/reinstall') && init?.method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { message: 'reinstall failed' } }), { status: 500 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers/gs-1') && init?.method === 'DELETE') {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { message: 'delete failed' } }), { status: 500 }),
        );
      }
      if (url.includes('/servers/srv-1/game-servers')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [gameServer] }), { status: 200 }));
      }
      if (url.includes('/servers/srv-1')) {
        return Promise.resolve(new Response(JSON.stringify(vpsServer), { status: 200 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    renderDetail();
    await waitFor(() => expect(screen.getByText('Forge RPG')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Удалить/ }));
    await user.click((await screen.findAllByRole('button', { name: /^OK$/i })).at(-1)!);
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('delete failed'));

    await user.click(screen.getByRole('button', { name: /Переустановить/i }));
    await user.click((await screen.findAllByRole('button', { name: /^OK$/i })).at(-1)!);
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('reinstall failed'));
  });
});
