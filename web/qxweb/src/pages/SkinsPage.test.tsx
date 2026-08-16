import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '@/test/test-utils';
import { SkinsPage } from './SkinsPage';
import { saveTokens, clearTokens } from '@/api/client';

vi.mock('skinview3d', async () => {
  const { skinview3dMock } = await import('@/test/skinview3d-mock');
  return skinview3dMock;
});

function mockSkinsFetch() {
  vi.mocked(fetch).mockImplementation((input: RequestInfo | URL) => {
    const url =
      typeof input === 'string'
        ? input
        : input instanceof URL
          ? input.href
          : input.url;
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
    if (url.includes('/users/me/cosmetics')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            skin_model: 'steve',
            has_skin: false,
            has_cape: false,
            updated_at: '2026-01-01T00:00:00Z',
          }),
          { status: 200 },
        ),
      );
    }
    if (url.includes('/cosmetics/skin-catalog')) {
      return Promise.resolve(
        new Response(JSON.stringify({ items: [] }), { status: 200 }),
      );
    }
    if (url.includes('/users/me')) {
      return Promise.resolve(
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
    }
    return Promise.resolve(new Response('{}', { status: 200 }));
  });
}

describe('SkinsPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
      saved_at: Date.now(),
    });
    mockSkinsFetch();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders skins management for authenticated users', async () => {
    renderWithProviders(<SkinsPage />, '/skins');

    await waitFor(() =>
      expect(screen.getByRole('heading', { name: /QXSkins для Minecraft/i })).toBeInTheDocument(),
    );
    expect(screen.getByText(/популярных источников/i)).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Загрузить скин/i })).toBeInTheDocument(),
    );
  });

  it('asks unauthenticated users to sign in', async () => {
    clearTokens();
    renderWithProviders(<SkinsPage />, '/skins');
    await waitFor(() =>
      expect(screen.getByText(/Войдите, чтобы управлять скинами/i)).toBeInTheDocument(),
    );
  });

  it('shows spinner while auth is loading', () => {
    vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));
    renderWithProviders(<SkinsPage />, '/skins');
    expect(document.querySelector('.ant-spin')).toBeTruthy();
  });
});
