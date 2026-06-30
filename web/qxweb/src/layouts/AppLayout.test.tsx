import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { AppLayout } from './AppLayout';
import * as BackendStatus from '@/backend/BackendStatusContext';
import { HomePage } from '@/pages/HomePage';
import { PlaceholderPage } from '@/pages/PlaceholderPage';
import { saveTokens } from '@/api/client';

describe('AppLayout', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.mocked(BackendStatus.useBackendStatus).mockReturnValue({ available: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows login navigation for unauthenticated users', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByText('QXSystem')).toBeInTheDocument());
    expect(screen.getAllByRole('button', { name: 'Вход' }).length).toBeGreaterThan(0);
  });

  it('disables login button when backend is unavailable', async () => {
    vi.mocked(BackendStatus.useBackendStatus).mockReturnValue({ available: false });

    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
    );

    await waitFor(() => expect(screen.getByRole('button', { name: 'Вход' })).toBeDisabled());
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
      expires_in: 3600,
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
    await user.click(screen.getByRole('radio', { name: 'Светлая тема' }));
    expect(window.localStorage.getItem('qxweb-theme')).toBe('light');
  });

  it('shows authenticated navigation and logs out', async () => {
    const user = userEvent.setup();
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
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
    await user.click(screen.getByRole('button', { name: 'US, Меню аккаунта' }));
    await user.click(await screen.findByText('Выйти'));
    await waitFor(() => expect(screen.queryByText('US')).not.toBeInTheDocument());
  });

  it('applies transparent home header that solidifies on scroll', async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>,
      '/',
    );

    const header = document.querySelector('header');
    expect(header?.className).toContain('app-header--landing');
    expect(header?.className).not.toContain('app-header--scrolled');

    Object.defineProperty(window, 'scrollY', { value: 32, configurable: true });
    window.dispatchEvent(new Event('scroll'));

    await waitFor(() => expect(header?.className).toContain('app-header--scrolled'));
  });

  it('uses landing sticky header on launcher route', () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="launcher" element={<PlaceholderPage title="L" phase="P1" />} />
        </Route>
      </Routes>,
      '/launcher',
    );

    const header = document.querySelector('header');
    expect(header?.className).toContain('app-header--landing');
    expect(header?.className).toContain('app-header--sticky');
  });
});
