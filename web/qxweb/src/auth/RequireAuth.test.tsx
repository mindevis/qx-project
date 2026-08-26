import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { RequireAuth } from '@/auth/RequireAuth';
import { saveTokens } from '@/api/client';

describe('RequireAuth', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('redirects guests to home', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<div>Home</div>} />
        <Route
          path="/servers"
          element={
            <RequireAuth>
              <div>Servers secret</div>
            </RequireAuth>
          }
        />
      </Routes>,
      '/servers',
    );

    await waitFor(() => expect(screen.getByText('Home')).toBeInTheDocument());
    expect(screen.queryByText('Servers secret')).not.toBeInTheDocument();
  });

  it('renders children when authenticated', async () => {
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

    renderWithProviders(
      <Routes>
        <Route path="/" element={<div>Home</div>} />
        <Route
          path="/skins"
          element={
            <RequireAuth>
              <div>Skins secret</div>
            </RequireAuth>
          }
        />
      </Routes>,
      '/skins',
    );

    await waitFor(() => expect(screen.getByText('Skins secret')).toBeInTheDocument());
  });
});
