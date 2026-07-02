import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api, saveTokens } from '@/api/client';
import { renderWithProviders } from '@/test/test-utils';
import { InstanceServerBinding } from './InstanceServerBinding';

const instance = {
  id: 'inst-1',
  name: 'Forge Client',
  mc_version: '1.21',
  loader: 'forge',
  created_at: 'now',
  updated_at: 'now',
};

const server = {
  id: 'mon-1',
  name: 'Survival World',
  server_type: 'forge',
  mc_version: '1.21',
  loader_version: '47.0.0',
  address: 'play.example.com',
  port: 25565,
  status: 'running',
  is_online: true,
  is_premium: false,
  description: '',
  banner_url: '',
  tags: [],
  mods: [],
  plugins: [],
  likes_count: 0,
  rating_avg: 0,
  rating_count: 0,
};

describe('InstanceServerBinding', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
      saved_at: Date.now(),
    });
    vi.mocked(fetch).mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.url;
      if (url.includes('/users/me')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ id: '1', email: 'user@test.com', tier: 'free', created_at: 'now' }),
            { status: 200 },
          ),
        );
      }
      return Promise.resolve(new Response('{}', { status: 200 }));
    });
    vi.spyOn(api, 'listBindableServers').mockResolvedValue({ items: [server] });
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'setMonitoringBinding').mockResolvedValue({
      game_server_id: 'mon-1',
      instance_id: 'inst-1',
      instance_name: 'Forge Client',
    });
    vi.spyOn(api, 'clearMonitoringBinding').mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  function renderBinding(variant: 'panel' | 'card' = 'panel') {
    return renderWithProviders(<InstanceServerBinding instance={instance} variant={variant} />);
  }

  it('loads bindable servers and saves binding', async () => {
    const user = userEvent.setup({ delay: null });
    renderBinding();
    await waitFor(() => expect(api.listBindableServers).toHaveBeenCalled());
    await user.click(screen.getByRole('combobox'));
    await user.click(await screen.findByText(/Survival World/));
    await waitFor(() =>
      expect(api.setMonitoringBinding).toHaveBeenCalledWith('mon-1', 'inst-1'),
    );
  });

  it('renders compact card variant', async () => {
    const { container } = renderBinding('card');
    await waitFor(() => expect(api.listBindableServers).toHaveBeenCalled());
    expect(container.querySelector('.qxmods-binding-panel--card')).toBeTruthy();
    expect(screen.queryByText(/Выберите свой игровой сервер/i)).not.toBeInTheDocument();
  });

  it('shows sign-in hint for guests', async () => {
    const { clearTokens } = await import('@/api/client');
    clearTokens();
    renderBinding();
    expect(await screen.findByText(/\u0412\u043e\u0439\u0434\u0438\u0442\u0435/i)).toBeInTheDocument();
  });

  it('clears binding when selection is cleared', async () => {
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({
      items: [{ game_server_id: 'mon-1', instance_id: 'inst-1', instance_name: 'Forge Client' }],
    });
    const user = userEvent.setup({ delay: null });
    renderBinding();
    await waitFor(() => expect(api.listBindableServers).toHaveBeenCalled());
    await user.click(screen.getByRole('combobox'));
    await user.click(await screen.findByRole('img', { name: 'close-circle' }));
    await waitFor(() => expect(api.clearMonitoringBinding).toHaveBeenCalledWith('mon-1'));
  });
});
