import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  api,
  checkBackendHealth,
  clearLinkedDevice,
  clearTokens,
  hasLauncherAccess,
  loadLinkedDevice,
  loadTokens,
  openServerConsole,
  saveLinkedDevice,
  saveTokens,
  wsBaseUrl,
  type TokenResponse,
} from './client';
import { logger } from '@/lib/logger';

const tokens: TokenResponse = {
  access_token: 'access',
  refresh_token: 'refresh',
  token_type: 'Bearer',
  expires_in: 3600,
};

describe('token storage', () => {
  it('loads null when storage is empty', () => {
    expect(loadTokens()).toBeNull();
  });

  it('loads null for invalid json', () => {
    localStorage.setItem('qx.auth', '{bad');
    expect(loadTokens()).toBeNull();
  });

  it('saves and clears tokens', () => {
    saveTokens(tokens);
    const loaded = loadTokens();
    expect(loaded?.access_token).toBe('access');
    expect(loaded?.saved_at).toBeTypeOf('number');
    clearTokens();
    expect(loadTokens()).toBeNull();
  });

  it('detects launcher access from user token', () => {
    expect(hasLauncherAccess()).toBe(false);
    saveTokens(tokens);
    expect(hasLauncherAccess()).toBe(true);
    clearTokens();
    expect(hasLauncherAccess()).toBe(false);
  });

  it('saves linked device id', () => {
    saveLinkedDevice('dev-1');
    expect(loadLinkedDevice()).toBe('dev-1');
    clearLinkedDevice();
    expect(loadLinkedDevice()).toBeNull();
  });
});

describe('api client', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('registers without auth header', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify(tokens), { status: 201 }),
    );

    const result = await api.register({
      email: 'a@b.com',
      password: 'password123',
    });

    expect(result.access_token).toBe('access');
    const [, init] = fetchMock.mock.calls[0];
    expect(init?.headers).not.toHaveProperty('Authorization');
  });

  it('sends bearer token for authenticated requests', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          id: '1',
          email: 'a@b.com',
          tier: 'free',
          created_at: 'now',
        }),
        { status: 200 },
      ),
    );

    await api.me();

    const [, init] = fetchMock.mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(headers.get('Authorization')).toBe('Bearer access');
  });

  it('refreshes access token before authenticated requests when close to expiry', async () => {
    localStorage.setItem(
      'qx.auth',
      JSON.stringify({
        ...tokens,
        expires_in: 60,
        saved_at: Date.now() - 59_000,
      }),
    );
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            access_token: 'access-new',
            refresh_token: 'refresh-new',
            token_type: 'Bearer',
            expires_in: 3600,
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'a@b.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );

    await api.me();

    expect(String(fetchMock.mock.calls[0][0])).toContain('/auth/refresh');
    const [, init] = fetchMock.mock.calls[1];
    const headers = new Headers(init?.headers);
    expect(headers.get('Authorization')).toBe('Bearer access-new');
  });

  it('retries authenticated requests once after refreshing on 401', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(new Response('unauthorized', { status: 401, statusText: 'Unauthorized' }))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            access_token: 'access-new',
            refresh_token: 'refresh-new',
            token_type: 'Bearer',
            expires_in: 3600,
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'a@b.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );

    await api.me();

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(String(fetchMock.mock.calls[1][0])).toContain('/auth/refresh');
  });

  it('returns undefined for 204 responses', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));

    await expect(api.logout()).resolves.toBeUndefined();
  });

  it('throws backend unavailable on network failure', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));

    await expect(api.login({ email: 'a@b.com', password: 'x' })).rejects.toMatchObject({
      code: 'BACKEND_UNAVAILABLE',
    });
  });

  it('throws backend unavailable on gateway errors', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(new Response('bad gateway', { status: 502, statusText: 'Bad Gateway' }));

    await expect(api.login({ email: 'a@b.com', password: 'x' })).rejects.toMatchObject({
      code: 'BACKEND_UNAVAILABLE',
    });
  });

  it('preserves CurseForge upstream errors instead of backend unavailable', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'CURSEFORGE_UNAVAILABLE',
            message: 'curseforge: status 403: invalid api key',
          },
        }),
        { status: 502, statusText: 'Bad Gateway' },
      ),
    );

    await expect(
      api.browseMods({ source: 'curseforge', loader: 'forge', mc_version: '1.21' }),
    ).rejects.toMatchObject({
      message: 'curseforge: status 403: invalid api key',
      apiCode: 'CURSEFORGE_UNAVAILABLE',
      code: undefined,
    });
  });

  it('preserves CONTENT_INSTALL_FAILED on 502', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'CONTENT_INSTALL_FAILED',
            message: 'download host not allowed',
          },
        }),
        { status: 502, statusText: 'Bad Gateway' },
      ),
    );

    await expect(
      api.syncPluginToGameServer('vps-1', 'gs-1', {
        source: 'spigot',
        project_id: '34315',
        project_name: 'Vault',
        version_id: '1',
        version_number: '1.7.3',
        filename: 'Vault.jar',
        download_url: 'https://cdn.spiget.org/file/spiget-resources/34315.jar',
      }),
    ).rejects.toMatchObject({
      message: 'download host not allowed',
      apiCode: 'CONTENT_INSTALL_FAILED',
      code: undefined,
    });
  });

  it('preserves SOURCE_UNAVAILABLE on 503', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'SOURCE_UNAVAILABLE',
            message: 'curseforge api key not configured',
          },
        }),
        { status: 503, statusText: 'Service Unavailable' },
      ),
    );

    await expect(
      api.browseMods({ source: 'curseforge', loader: 'forge', mc_version: '1.21' }),
    ).rejects.toMatchObject({
      message: 'curseforge api key not configured',
      apiCode: 'SOURCE_UNAVAILABLE',
      code: undefined,
    });
  });

  it('maps catalog request abort to upstream unavailable', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockRejectedValue(new DOMException('Aborted', 'AbortError'));

    await expect(api.browseMods({ type: 'mod' })).rejects.toMatchObject({
      apiCode: 'UPSTREAM_UNAVAILABLE',
    });
    expect(fetchMock.mock.calls[0]?.[1]).toEqual(expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it('throws api error message when present', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({ error: { code: 'X', message: 'bad request' } }),
        { status: 400, statusText: 'Bad Request' },
      ),
    );

    await expect(api.login({ email: 'a@b.com', password: 'x' })).rejects.toThrow(
      'bad request',
    );
  });

  it('falls back to status text when api error has no message', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'X' } }), {
        status: 400,
        statusText: 'Bad Request',
      }),
    );

    await expect(api.login({ email: 'a@b.com', password: 'x' })).rejects.toThrow('Bad Request');
  });

  it('falls back to status text when body is not json', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(new Response('nope', { status: 500, statusText: 'Server Error' }));

    await expect(api.login({ email: 'a@b.com', password: 'x' })).rejects.toThrow('Server Error');
  });

  it('changes password and email', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'new@b.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );

    await expect(
      api.changePassword({ current_password: 'old', new_password: 'newpass12' }),
    ).resolves.toBeUndefined();

    const profile = await api.changeEmail({
      current_password: 'old',
      email: 'new@b.com',
    });
    expect(profile.email).toBe('new@b.com');
  });

  it('loads linked launcher device for user', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          linked: true,
          device_id: 'dev-1',
          status: 'linked',
          owner_type: 'user',
        }),
        { status: 200 },
      ),
    );

    const device = await api.myLauncherDevice();
    expect(device.linked).toBe(true);
    expect(device.device_id).toBe('dev-1');
  });

  it('unlinks launcher device', async () => {
    saveTokens({
      access_token: 'access',
      refresh_token: 'refresh',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: 'pending_link' }), { status: 200 }),
    );

    const result = await api.unlinkDevice();
    expect(result.status).toBe('pending_link');
    const [, init] = vi.mocked(fetch).mock.calls[0]!;
    expect(init?.method).toBe('POST');
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access');
  });

  it('starts, stops, and restarts servers', async () => {
    saveTokens({
      access_token: 'access',
      refresh_token: 'refresh',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch)
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'online' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'offline' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'online' }), { status: 200 }));

    await expect(api.startServer('srv-1')).resolves.toEqual({ status: 'online' });
    await expect(api.stopServer('srv-1')).resolves.toEqual({ status: 'offline' });
    await expect(api.restartServer('srv-1')).resolves.toEqual({ status: 'online' });

    const methods = vi.mocked(fetch).mock.calls.map(([, init]) => init?.method);
    expect(methods).toEqual(['POST', 'POST', 'POST']);
  });

  it('uses user bearer for launcher api', async () => {
    saveTokens(tokens);
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );

    await api.listProfiles();

    const [, init] = vi.mocked(fetch).mock.calls[0]!;
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access');
  });

  it('omits authorization header when access token is empty', async () => {
    saveTokens({
      access_token: '',
      refresh_token: 'refresh',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );

    await api.listServers();

    const [, init] = vi.mocked(fetch).mock.calls[0]!;
    expect(new Headers(init?.headers).get('Authorization')).toBeNull();
  });

  it('calls launcher endpoints without auth when not signed in', async () => {
    clearTokens();
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'p1',
            username: 'Steve',
            offline_uuid: 'uuid',
            created_at: 't',
          }),
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ items: [{ id: 'p1', username: 'Steve', offline_uuid: 'uuid', created_at: 't' }] }), {
          status: 200,
        }),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'lr-1',
            status: 'queued',
            instance_id: 'inst-1',
            expires_at: '2099-01-01T00:00:00Z',
          }),
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'lr-1',
            status: 'running',
            instance_id: 'inst-1',
            expires_at: '2099-01-01T00:00:00Z',
          }),
          { status: 200 },
        ),
      );

    const created = await api.createProfile({ username: 'Steve' });
    expect(created.username).toBe('Steve');

    const list = await api.listProfiles();
    expect(list.items).toHaveLength(1);

    await api.deleteProfile('p1');

    const launch = await api.createLaunchRequest({
      instance_id: 'inst-1',
      offline_profile_id: 'p1',
      join_server_address: 'play.example.com',
      join_server_port: 25565,
    });
    expect(launch.status).toBe('queued');

    const status = await api.getLaunchRequest('lr-1');
    expect(status.status).toBe('running');
  });

  it('opens server console websocket', () => {
    saveTokens(tokens);
    const sent: string[] = [];
    const onMessage = vi.fn();
    const onClose = vi.fn();
    const instances: MockWS[] = [];
    class MockWS {
      static OPEN = 1;
      static CONNECTING = 0;
      readyState = MockWS.OPEN;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      onmessage: ((ev: { data: string }) => void) | null = null;
      close = vi.fn(function (this: MockWS) {
        this.onclose?.();
      });
      constructor(public url: string) {
        expect(url).toContain('/servers/s1/console');
        expect(url).toContain('access_token=access');
        instances.push(this);
      }
      send(data: string) {
        sent.push(data);
      }
    }
    vi.stubGlobal('WebSocket', MockWS);

    const session = openServerConsole('s1', { onMessage, onClose });
    session.send('list');
    expect(sent[0]).toContain('list');

    instances[0]?.onmessage?.({ data: '{bad json' });
    instances[0]?.onmessage?.({ data: JSON.stringify({ type: 'output', line: 'ok' }) });
    expect(onMessage).toHaveBeenCalledWith({ type: 'output', line: 'ok' });
    instances[0]?.onerror?.();

    session.close();
    expect(onClose).toHaveBeenCalled();
  });

  it('closes open websocket on client disconnect', () => {
    saveTokens(tokens);
    const close = vi.fn();
    class MockWS {
      static OPEN = 1;
      static CONNECTING = 0;
      readyState = MockWS.OPEN;
      close = close;
      constructor(_url: string) {}
    }
    vi.stubGlobal('WebSocket', MockWS);
    const session = openServerConsole('s1', { onMessage: vi.fn() });
    session.close();
    expect(close).toHaveBeenCalled();
  });

  it('does not log websocket errors after client close', () => {
    saveTokens(tokens);
    const warnSpy = vi.spyOn(logger, 'warn');
    const instances: Array<{ onerror: (() => void) | null }> = [];
    class MockWS {
      static OPEN = 1;
      static CONNECTING = 0;
      readyState = MockWS.OPEN;
      onerror: (() => void) | null = null;
      close = vi.fn();
      constructor(_url: string) {
        instances.push(this);
      }
    }
    vi.stubGlobal('WebSocket', MockWS);

    const session = openServerConsole('s1', { onMessage: vi.fn() });
    session.close();
    instances[0]?.onerror?.();
    expect(warnSpy).not.toHaveBeenCalled();
  });

  it('skips close when websocket is already closed', () => {
    saveTokens(tokens);
    const close = vi.fn();
    class MockWS {
      static OPEN = 1;
      static CONNECTING = 0;
      readyState = 2;
      close = close;
      constructor(_url: string) {}
    }
    vi.stubGlobal('WebSocket', MockWS);
    const session = openServerConsole('s1', { onMessage: vi.fn() });
    session.close();
    expect(close).not.toHaveBeenCalled();
  });

  it('defers close while websocket is still connecting', () => {
    saveTokens(tokens);
    const close = vi.fn();
    class MockWS {
      static OPEN = 1;
      static CONNECTING = 0;
      readyState = MockWS.CONNECTING;
      close = close;
      addEventListener(type: string, fn: () => void, _opts?: { once?: boolean }) {
        if (type === 'open') fn();
      }
      constructor(_url: string) {}
    }
    vi.stubGlobal('WebSocket', MockWS);
    const session = openServerConsole('s1', { onMessage: vi.fn() });
    session.close();
    expect(close).toHaveBeenCalled();
  });

  it('does not send when websocket is not open', () => {
    saveTokens(tokens);
    const send = vi.fn();
    class MockWS {
      static OPEN = 1;
      readyState = 0;
      close = vi.fn();
      send = send;
      constructor(_url: string) {}
    }
    vi.stubGlobal('WebSocket', MockWS);
    const session = openServerConsole('s1', { onMessage: vi.fn() });
    session.send('noop');
    expect(send).not.toHaveBeenCalled();
  });

  it('builds websocket url from absolute api base', () => {
    expect(wsBaseUrl('https://api.example.com/api/v1')).toBe('wss://api.example.com');
    expect(wsBaseUrl('http://localhost:3000/api/v1')).toBe('ws://localhost:3000');
    expect(wsBaseUrl('/api/v1')).toMatch(/^ws:\/\//);
  });

  it('builds relative websocket url with wss on https pages', () => {
    vi.stubGlobal('location', { ...window.location, protocol: 'https:', host: 'qx.example.com' });
    expect(wsBaseUrl('/api/v1')).toBe('wss://qx.example.com');
    vi.unstubAllGlobals();
    vi.stubGlobal('fetch', vi.fn());
  });

  it('opens console websocket without stored token', () => {
    clearTokens();
    const instances: MockWS[] = [];
    class MockWS {
      static OPEN = 1;
      readyState = MockWS.OPEN;
      close = vi.fn();
      constructor(public url: string) {
        expect(url).toContain('/servers/s1/console');
        expect(url).not.toContain('access_token=');
        instances.push(this);
      }
      send() {}
    }
    vi.stubGlobal('WebSocket', MockWS);
    openServerConsole('s1', { onMessage: vi.fn() });
    expect(instances).toHaveLength(1);
  });

  it('checkBackendHealth returns true for ok response', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok' }), { status: 200 }),
    );

    await expect(checkBackendHealth()).resolves.toBe(true);
  });

  it('checkBackendHealth returns false for failed response', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(checkBackendHealth()).resolves.toBe(false);
  });

  it('checkBackendHealth returns false when fetch throws', async () => {
    const warn = vi.spyOn(logger, 'warn');
    vi.mocked(fetch).mockRejectedValue(new Error('network down'));

    await expect(checkBackendHealth()).resolves.toBe(false);
    expect(warn).toHaveBeenCalledWith('backend health check failed');
  });

  it('calls dedicated server game server management endpoints', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    const gameServer = {
      id: 'gs-1',
      name: 'Test',
      server_type: 'forge',
      mc_version: '1.20.1',
      port: 25565,
      status: 'running',
      created_at: 'now',
    };

    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(gameServer), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ properties: [] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'ok' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ content: 'line' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'ok' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'ok', path: 'plugins' }), { status: 201 }));

    await api.updateVpsGameServer('srv-1', 'gs-1', { name: 'Renamed' });
    await api.deleteVpsGameServer('srv-1', 'gs-1');
    await api.cloneVpsGameServer('srv-1', 'gs-1');
    await api.changeVpsGameServerVersion('srv-1', 'gs-1', {
      mc_version: '1.21.1',
      loader_version: '20',
    });
    await api.reinstallVpsGameServer('srv-1', 'gs-1');
    await api.startVpsGameServer('srv-1', 'gs-1');
    await api.stopVpsGameServer('srv-1', 'gs-1');
    await api.restartVpsGameServer('srv-1', 'gs-1');
    await api.getVpsGameServer('srv-1', 'gs-1');
    await api.getVpsGameServerProperties('srv-1', 'gs-1');
    await api.patchVpsGameServerProperties('srv-1', 'gs-1', { 'max-players': '20' });
    await api.listGameServerResources('srv-1', 'gs-1', { kind: 'mod' });
    await api.listVpsGameServerMods('srv-1', 'gs-1');
    await api.listVpsGameServerFiles('srv-1', 'gs-1', 'config');
    await api.readVpsGameServerFile('srv-1', 'gs-1', 'server.properties');
    await api.writeVpsGameServerFile('srv-1', 'gs-1', 'server.properties', 'motd=hi');
    await api.mkdirVpsGameServerFile('srv-1', 'gs-1', 'plugins');

    const urls = fetchMock.mock.calls.map(([input]) => String(input));
    expect(urls.some((url) => url.includes('/game-servers/gs-1/clone'))).toBe(true);
    expect(urls.some((url) => url.includes('/game-servers/gs-1/version'))).toBe(true);
    expect(urls.some((url) => url.includes('/game-servers/gs-1') && url.includes('properties'))).toBe(
      true,
    );
    expect(urls.some((url) => url.includes('/files/content'))).toBe(true);
    expect(urls.some((url) => url.includes('/files/mkdir'))).toBe(true);
  });

  it('calls cosmetics, mojang, monitoring, and mods endpoints', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    const cosmetics = {
      skin_model: 'steve' as const,
      has_skin: false,
      has_cape: false,
      updated_at: '2026-01-01T00:00:00Z',
    };
    const monitoringServer = {
      id: 'mon-1',
      name: 'Test',
      server_type: 'forge',
      mc_version: '1.21',
      address: '127.0.0.1',
      port: 25565,
      status: 'running',
      is_online: true,
      is_premium: false,
      tags: [] as string[],
      mods: [] as string[],
      plugins: [] as string[],
      likes_count: 0,
      rating_avg: 0,
      rating_count: 0,
    };

    fetchMock.mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? 'GET';
      if (url.includes('/users/me/mojang/oauth/start')) {
        return Promise.resolve(
          new Response(JSON.stringify({ authorization_url: 'https://oauth.test' }), { status: 200 }),
        );
      }
      if (url.includes('/users/me/mojang') && method === 'DELETE') {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      if (url.includes('/users/me/mojang')) {
        return Promise.resolve(new Response(JSON.stringify({ linked: false }), { status: 200 }));
      }
      if (url.includes('/users/me/cosmetics/skin') || url.includes('/users/me/cosmetics/cape')) {
        return Promise.resolve(new Response(JSON.stringify(cosmetics), { status: 200 }));
      }
      if (url.includes('/users/me/cosmetics')) {
        return Promise.resolve(new Response(JSON.stringify(cosmetics), { status: 200 }));
      }
      if (url.includes('/monitoring/servers') && url.includes('/like')) {
        return Promise.resolve(new Response(JSON.stringify(monitoringServer), { status: 200 }));
      }
      if (url.includes('/monitoring/servers') && url.includes('/rate')) {
        return Promise.resolve(new Response(JSON.stringify(monitoringServer), { status: 200 }));
      }
      if (url.includes('/monitoring/servers')) {
        return Promise.resolve(
          new Response(JSON.stringify({ items: [monitoringServer] }), { status: 200 }),
        );
      }
      if (url.includes('/mods/search')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              items: [
                {
                  source: 'modrinth',
                  id: 'sodium',
                  name: 'Sodium',
                  summary: 'Fast',
                  external_url: 'https://modrinth.com/mod/sodium',
                },
              ],
              curseforge_enabled: true,
            }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/mods/modrinth/sodium/versions')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              items: [
                {
                  id: 'ver-1',
                  version_number: '1.0',
                  files: [{ filename: 'sodium.jar', url: 'https://example/mod.jar' }],
                },
              ],
            }),
            { status: 200 },
          ),
        );
      }
      if (url.includes('/mods/sync')) {
        return Promise.resolve(new Response(JSON.stringify({ status: 'queued' }), { status: 200 }));
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });

    await api.mojangStatus();
    await api.startMojangOAuth();
    await api.unlinkMojang();
    await api.getCosmetics();
    await api.updateCosmetics({ skin_model: 'alex' });
    await api.deleteCosmeticsSkin();
    await api.deleteCosmeticsCape();
    const file = new File(['png'], 'skin.png', { type: 'image/png' });
    await api.uploadCosmeticsSkin(file);
    await api.uploadCosmeticsCape(file);
    await api.listMonitoringServers({ q: 'test', loader: 'forge' });
    await api.likeMonitoringServer('mon-1');
    await api.rateMonitoringServer('mon-1', 5);
    await api.listMonitoringBindings();
    await api.setMonitoringBinding('mon-1', 'inst-1');
    await api.clearMonitoringBinding('mon-1');
    await api.searchMods({ q: 'sodium', loader: 'forge', mc_version: '1.21' });
    await api.listModVersions('modrinth', 'sodium', { loader: 'forge', mc_version: '1.21' });
    await api.syncModToGameServer('srv-1', 'gs-1', {
      source: 'modrinth',
      project_id: 'sodium',
      version_id: 'ver-1',
      filename: 'sodium.jar',
      download_url: 'https://example/mod.jar',
    });

    const urls = fetchMock.mock.calls.map(([input]) => String(input));
    expect(urls.some((url) => url.includes('/users/me/mojang'))).toBe(true);
    expect(urls.some((url) => url.includes('/users/me/cosmetics/skin'))).toBe(true);
    expect(urls.some((url) => url.includes('/monitoring/servers'))).toBe(true);
    expect(urls.some((url) => url.includes('/mods/search'))).toBe(true);
  });
});
