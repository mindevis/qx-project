import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { AppLayout } from './AppLayout';
import { HomePage } from '@/pages/HomePage';
import { saveTokens } from '@/api/client';

describe('AppLayout', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows guest navigation', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('QXProject')).toBeInTheDocument());
    expect(screen.getAllByRole('button', { name: 'Вход' }).length).toBeGreaterThan(0);
  });

  it('shows header spinner while auth is loading', () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    expect(document.querySelector('header .ant-spin')).toBeTruthy();
  });

  it('shows servers link in dark theme', async () => {
    const user = userEvent.setup({ delay: null });
    window.localStorage.setItem('qxweb-theme', 'dark');
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockResolvedValueOnce(
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
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('Серверы')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Светлая тема' }));
    expect(window.localStorage.getItem('qxweb-theme')).toBe('light');
  });

  it('shows authenticated navigation and logs out', async () => {
    const user = userEvent.setup();
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'user@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('US')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Меню аккаунта' }));
    await user.click(await screen.findByText('Выйти'));
    await waitFor(() => expect(screen.queryByText('US')).not.toBeInTheDocument());
  });
});
