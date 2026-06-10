import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  api,
  clearTokens,
  loadTokens,
  saveTokens,
  type TokenResponse,
} from './client';

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
});
