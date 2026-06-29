import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { api, saveTokens } from '@/api/client';
import { renderWithProviders } from '@/test/test-utils';
import { MonitoringPage } from './MonitoringPage';

const sampleServer = {
  id: 'mon-1',
  name: 'Survival World',
  server_type: 'forge',
  mc_version: '1.21',
  loader_version: '47.0.0',
  address: 'play.example.com',
  port: 25565,
  status: 'running',
  is_online: true,
  is_premium: true,
  description: 'Best survival server',
  banner_url: 'https://example.com/banner.png',
  tags: ['survival'],
  mods: ['jei'],
  plugins: [],
  likes_count: 3,
  rating_avg: 4.2,
  rating_count: 5,
};

function mockAuthedFetch() {
  return (input: RequestInfo | URL) => {
    const url =
      typeof input === 'string'
        ? input
        : input instanceof URL
          ? input.href
          : input.url;
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
  };
}

describe('MonitoringPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'listMcVersions').mockResolvedValue({
      items: [{ id: '1.21', type: 'release' }],
    });
    vi.spyOn(api, 'listMonitoringServers').mockResolvedValue({ items: [sampleServer] });
    vi.spyOn(api, 'likeMonitoringServer').mockResolvedValue({
      ...sampleServer,
      likes_count: 4,
    });
    vi.spyOn(api, 'rateMonitoringServer').mockResolvedValue({
      ...sampleServer,
      rating_avg: 5,
      rating_count: 6,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('lists monitoring servers', async () => {
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    expect(screen.getByText('Premium')).toBeInTheDocument();
    expect(screen.getByText('play.example.com:25565')).toBeInTheDocument();
  });

  it('shows empty state when catalog is empty', async () => {
    vi.mocked(api.listMonitoringServers).mockResolvedValueOnce({ items: [] });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() =>
      expect(screen.getByText('Пока нет серверов в мониторинге')).toBeInTheDocument(),
    );
  });

  it('likes a server when authenticated', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(mockAuthedFetch());
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await user.click(document.querySelector('.monitoring-card-like')!);
    await waitFor(() => expect(api.likeMonitoringServer).toHaveBeenCalledWith('mon-1'));
    expect(message.success).toHaveBeenCalledWith('Лайк добавлен');
  });

  it('opens auth modal when guest tries to like', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await user.click(document.querySelector('.monitoring-card-like')!);
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  });

  it('applies search filters', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Поиск по названию, описанию или тегам'), 'survival');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() =>
      expect(api.listMonitoringServers).toHaveBeenLastCalledWith(
        expect.objectContaining({ q: 'survival' }),
      ),
    );
  });
});
