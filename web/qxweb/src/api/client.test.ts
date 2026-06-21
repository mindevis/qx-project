import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  api,
  checkBackendHealth,
  clearGuestSession,
  clearLinkedDevice,
  clearTokens,
  hasLauncherAccess,
  loadGuestSession,
  loadLinkedDevice,
  loadTokens,
  openServerConsole,
  saveGuestSession,
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
  expires_in: 60,
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
    expect(loadTokens()).toEqual(tokens);
    clearTokens();
    expect(loadTokens()).toBeNull();
  });

  it('saves and clears guest session', () => {
    saveGuestSession({ guest_token: 'g', expires_in: 3600 });
    expect(loadGuestSession()?.guest_token).toBe('g');
    expect(hasLauncherAccess()).toBe(true);
    clearGuestSession();
    expect(loadGuestSession()).toBeNull();
    expect(hasLauncherAccess()).toBe(false);
  });

  it('detects launcher access from user or guest token', () => {
    saveTokens(tokens);
    expect(hasLauncherAccess()).toBe(true);
    clearTokens();
    saveGuestSession({ guest_token: 'g', expires_in: 3600 });
    expect(hasLauncherAccess()).toBe(true);
    saveTokens({ ...tokens, access_token: '' });
    expect(hasLauncherAccess()).toBe(true);
  });

  it('loads null guest session for invalid json', () => {
    localStorage.setItem('qx.guest', '{bad');
    expect(loadGuestSession()).toBeNull();
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

  it('returns undefined for 204 responses', async () => {
    saveTokens(tokens);
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));

    await expect(api.logout()).resolves.toBeUndefined();
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

  it('uses guest bearer or omits auth for launcher api', async () => {
    clearTokens();
    clearGuestSession();
    saveGuestSession({ guest_token: 'guest-token', expires_in: 3600 });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );

    await api.listProfiles();

    let [, init] = vi.mocked(fetch).mock.calls[0]!;
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer guest-token');

    clearGuestSession();
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );
    await api.listProfiles();
    [, init] = vi.mocked(fetch).mock.calls[1]!;
    expect(new Headers(init?.headers).get('Authorization')).toBeNull();
  });

  it('calls launcher endpoints without auth when session is missing', async () => {
    clearTokens();
    clearGuestSession();
    saveGuestSession({ guest_token: 'guest', expires_in: 3600 });
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

    const launch = await api.createLaunchRequest({ instance_id: 'inst-1', offline_profile_id: 'p1' });
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
    warnSpy.mockRestore();
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
        expect(url).toContain('access_token=');
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
});
