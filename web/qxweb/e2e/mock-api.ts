import type { Page, Route } from '@playwright/test';

type LauncherInstance = {
  id: string;
  name: string;
  mc_version: string;
  loader: string;
  created_at: string;
  updated_at: string;
};

type OfflineProfile = {
  id: string;
  username: string;
  offline_uuid: string;
  model?: 'steve' | 'alex';
  created_at: string;
};

type LaunchRequest = {
  id: string;
  status: string;
  instance_id: string;
  offline_profile_id?: string;
  expires_at: string;
};

export type MockApiState = {
  instances: LauncherInstance[];
  profiles: OfflineProfile[];
  launchRequests: Map<string, LaunchRequest>;
  linkedDevice: { device_id: string; owner_type: 'user' } | null;
};

export function createMockState(): MockApiState {
  return {
    instances: [],
    profiles: [],
    launchRequests: new Map(),
    linkedDevice: null,
  };
}

function isAuthUser(request: { headers: () => Record<string, string> }): boolean {
  const auth = request.headers()['authorization'] ?? '';
  return auth === 'Bearer access-e2e';
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

export async function installLauncherApiMock(page: Page, state: MockApiState) {
  await page.route('**/api/v1/**', async (route) => {
    const path = apiPath(route.request().url());
    const method = route.request().method();

    if (path === '/instances' && method === 'GET') {
      return json(route, 200, { items: state.instances });
    }
    if (path === '/instances' && method === 'POST') {
      const body = route.request().postDataJSON() as {
        name: string;
        mc_version: string;
        loader?: string;
      };
      const inst: LauncherInstance = {
        id: `inst-${state.instances.length + 1}`,
        name: body.name,
        mc_version: body.mc_version,
        loader: body.loader ?? 'vanilla',
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:00Z',
      };
      state.instances.push(inst);
      return json(route, 201, inst);
    }
    if (path.startsWith('/instances/') && method === 'DELETE') {
      const id = path.split('/')[2];
      state.instances = state.instances.filter((i) => i.id !== id);
      return route.fulfill({ status: 204, body: '' });
    }

    if (path === '/launcher/profiles' && method === 'GET') {
      return json(route, 200, { items: state.profiles });
    }
    if (path === '/launcher/profiles' && method === 'POST') {
      const body = route.request().postDataJSON() as { username: string; model?: 'steve' | 'alex' };
      const profile: OfflineProfile = {
        id: `prof-${state.profiles.length + 1}`,
        username: body.username,
        offline_uuid: '00000000-0000-0000-0000-000000000001',
        model: body.model === 'alex' ? 'alex' : 'steve',
        created_at: '2026-06-10T00:00:00Z',
      };
      state.profiles.push(profile);
      return json(route, 201, profile);
    }

    if (path === '/launcher/devices/link' && method === 'POST') {
      const body = route.request().postDataJSON() as { device_id: string };
      if (!isAuthUser(route.request())) {
        return json(route, 401, { error: { code: 'UNAUTHORIZED', message: 'auth required' } });
      }
      state.linkedDevice = { device_id: body.device_id, owner_type: 'user' };
      return json(route, 200, {
        status: 'linked',
        owner_type: 'user',
      });
    }

    if (path === '/launcher/launch-requests' && method === 'POST') {
      const body = route.request().postDataJSON() as {
        instance_id: string;
        offline_profile_id?: string;
      };
      const id = `lr-${state.launchRequests.size + 1}`;
      const req: LaunchRequest = {
        id,
        status: 'queued',
        instance_id: body.instance_id,
        offline_profile_id: body.offline_profile_id,
        expires_at: '2026-06-10T01:00:00Z',
      };
      state.launchRequests.set(id, req);
      return json(route, 201, req);
    }
    if (path.startsWith('/launcher/launch-requests/') && method === 'GET') {
      const id = path.split('/')[3];
      const req = state.launchRequests.get(id);
      if (!req) {
        return json(route, 404, { error: { code: 'NOT_FOUND', message: 'not found' } });
      }
      req.status = 'completed';
      state.launchRequests.set(id, req);
      return json(route, 200, req);
    }

    if (path === '/users/me' && method === 'GET') {
      return json(route, 200, {
        id: 'user-1',
        email: 'e2e@test.com',
        tier: 'free',
        created_at: '2026-06-10T00:00:00Z',
      });
    }
    if (path === '/users/me/launcher-device' && method === 'GET') {
      if (state.linkedDevice?.owner_type === 'user') {
        return json(route, 200, {
          linked: true,
          device_id: state.linkedDevice.device_id,
          status: 'linked',
        });
      }
      return json(route, 200, { linked: false });
    }

    if (path === '/launcher/mc-versions' && method === 'GET') {
      return json(route, 200, {
        latest: { release: '1.21' },
        items: [
          { id: '1.21', type: 'release' },
          { id: '1.20.4', type: 'release' },
        ],
      });
    }

    return json(route, 404, { error: { code: 'NOT_FOUND', message: `unmocked ${method} ${path}` } });
  });
}

export function seedAuthSession(page: Page) {
  return page.addInitScript(() => {
    localStorage.setItem(
      'qx.auth',
      JSON.stringify({
        access_token: 'access-e2e',
        refresh_token: 'refresh-e2e',
        token_type: 'Bearer',
        expires_in: 3600,
      }),
    );
  });
}
