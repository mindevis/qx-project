import type { Page, Route } from '@playwright/test';

export type GameServer = {
  id: string;
  name: string;
  slug: string;
  server_type: string;
  status: string;
  mc_version?: string;
  config: {
    jar_path?: string;
    jvm_args?: string[];
    extra_args?: string[];
  };
  ssh: {
    host: string;
    port: number;
    username: string;
  };
  agent_deployed: boolean;
  agent_online: boolean;
  minecraft_running?: boolean;
  created_at: string;
  updated_at: string;
};

export type ServersMockState = {
  servers: Map<string, GameServer>;
};

export function createServersMockState(): ServersMockState {
  return { servers: new Map() };
}

function apiPath(url: string): string {
  const u = new URL(url);
  const idx = u.pathname.indexOf('/api/v1');
  return idx >= 0 ? u.pathname.slice(idx + '/api/v1'.length) : u.pathname;
}

function json(route: Route, status: number, body: unknown) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

function isAuthUser(request: { headers: () => Record<string, string> }): boolean {
  const auth = request.headers()['authorization'] ?? '';
  return auth === 'Bearer access-e2e';
}

function cloneServer(server: GameServer): GameServer {
  return {
    ...server,
    config: { ...server.config },
    ssh: { ...server.ssh },
  };
}

export async function installServersApiMock(page: Page, state: ServersMockState) {
  await page.route('**/api/v1/**', async (route) => {
    const path = apiPath(route.request().url());
    const method = route.request().method();

    if (!isAuthUser(route.request()) && path !== '/users/me') {
      return json(route, 401, { error: { code: 'UNAUTHORIZED', message: 'unauthorized' } });
    }

    if (path === '/users/me' && method === 'GET') {
      return json(route, 200, {
        id: 'user-1',
        email: 'e2e@test.com',
        tier: 'free',
        created_at: '2026-06-10T00:00:00Z',
      });
    }

    if (path === '/servers' && method === 'GET') {
      return json(route, 200, { items: [...state.servers.values()] });
    }

    if (path === '/servers' && method === 'POST') {
      const body = route.request().postDataJSON() as {
        name: string;
        mc_version?: string;
        ssh: { host: string; port?: number; username: string };
        config?: { jar_path?: string };
      };
      const id = `srv-${state.servers.size + 1}`;
      const server: GameServer = {
        id,
        name: body.name,
        slug: body.name.toLowerCase().replace(/\s+/g, '-'),
        server_type: 'vanilla',
        status: 'pending',
        mc_version: body.mc_version,
        config: { jar_path: body.config?.jar_path, jvm_args: [], extra_args: [] },
        ssh: {
          host: body.ssh.host,
          port: body.ssh.port ?? 22,
          username: body.ssh.username,
        },
        agent_deployed: false,
        agent_online: false,
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:00Z',
      };
      state.servers.set(id, server);
      return json(route, 201, cloneServer(server));
    }

    const serverMatch = path.match(/^\/servers\/([^/]+)(?:\/(.+))?$/);
    if (serverMatch) {
      const id = serverMatch[1];
      const action = serverMatch[2];
      const server = state.servers.get(id);
      if (!server) {
        return json(route, 404, { error: { code: 'NOT_FOUND', message: 'not found' } });
      }

      if (!action && method === 'GET') {
        return json(route, 200, cloneServer(server));
      }
      if (!action && method === 'DELETE') {
        state.servers.delete(id);
        return route.fulfill({ status: 204, body: '' });
      }
      if (action === 'deploy' && method === 'POST') {
        server.status = 'offline';
        server.agent_deployed = true;
        server.agent_online = true;
        server.updated_at = '2026-06-10T00:01:00Z';
        state.servers.set(id, server);
        return json(route, 200, cloneServer(server));
      }
      if (action === 'start' && method === 'POST') {
        server.status = 'online';
        server.minecraft_running = true;
        server.updated_at = '2026-06-10T00:02:00Z';
        state.servers.set(id, server);
        return json(route, 200, { status: 'online' });
      }
      if (action === 'stop' && method === 'POST') {
        server.status = 'offline';
        server.minecraft_running = false;
        server.updated_at = '2026-06-10T00:03:00Z';
        state.servers.set(id, server);
        return json(route, 200, { status: 'offline' });
      }
      if (action === 'restart' && method === 'POST') {
        return json(route, 200, { status: 'online' });
      }
    }

    return json(route, 404, { error: { code: 'NOT_FOUND', message: `unmocked ${method} ${path}` } });
  });
}

export { seedAuthSession } from './mock-api';
