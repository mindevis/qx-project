import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { api, saveTokens } from '@/api/client';
import { renderWithProviders } from '@/test/test-utils';
import { testNavigation } from '@/test/navigation-mock';
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
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'setMonitoringBinding').mockResolvedValue({
      game_server_id: 'mon-1',
      instance_id: 'inst-1',
      instance_name: 'Forge Client',
    });
    vi.spyOn(api, 'clearMonitoringBinding').mockResolvedValue(undefined);
    vi.spyOn(api, 'listInstances').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'myLauncherDevice').mockRejectedValue(new Error('no device'));
    vi.spyOn(api, 'listProfiles').mockResolvedValue({ items: [] });
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
    expect(testMessage.success).toHaveBeenCalledWith('Лайк добавлен');
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

  it('saves instance binding when authenticated', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(mockAuthedFetch());
    vi.spyOn(api, 'listInstances').mockResolvedValue({
      items: [
        {
          id: 'inst-1',
          name: 'Forge Client',
          mc_version: '1.21',
          loader: 'forge',
          created_at: 'now',
          updated_at: 'now',
        },
      ],
    });
    vi.spyOn(api, 'setMonitoringBinding').mockResolvedValue({
      game_server_id: 'mon-1',
      instance_id: 'inst-1',
      instance_name: 'Forge Client',
    });

    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await waitFor(() =>
      expect(screen.getByRole('combobox', { name: 'Инстанс лаунчера' })).toBeInTheDocument(),
    );

    await user.click(screen.getByRole('combobox', { name: 'Инстанс лаунчера' }));
    await user.click(await screen.findByText('Forge Client (1.21)'));

    await waitFor(() =>
      expect(api.setMonitoringBinding).toHaveBeenCalledWith('mon-1', 'inst-1'),
    );
    expect(testMessage.success).toHaveBeenCalledWith('Привязка инстанса сохранена');
  });

  it('does not launch via QXLauncher when no binding is set', async () => {
    const createSpy = vi.spyOn(api, 'createLaunchRequest');
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Подключиться/ }));
    expect(testNavigation.hrefSet).toHaveBeenCalledWith('minecraft://play.example.com:25565');
    expect(createSpy).not.toHaveBeenCalled();
  });

  it('clears instance binding when authenticated', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(mockAuthedFetch());
    vi.spyOn(api, 'listInstances').mockResolvedValue({
      items: [
        {
          id: 'inst-1',
          name: 'Forge Client',
          mc_version: '1.21',
          loader: 'forge',
          created_at: 'now',
          updated_at: 'now',
        },
      ],
    });
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({
      items: [{ game_server_id: 'mon-1', instance_id: 'inst-1', instance_name: 'Forge Client' }],
    });

    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await waitFor(() =>
      expect(screen.getByRole('combobox', { name: 'Инстанс лаунчера' })).toBeInTheDocument(),
    );

    const card = screen.getByText('Survival World').closest('article');
    expect(card).not.toBeNull();
    await user.click(within(card!).getByRole('img', { name: 'close-circle' }));

    await waitFor(() => expect(api.clearMonitoringBinding).toHaveBeenCalledWith('mon-1'));
    expect(testMessage.success).toHaveBeenCalledWith('Привязка инстанса удалена');
  });

  it('launches bound instance when launcher is linked', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(mockAuthedFetch());
    vi.spyOn(api, 'listInstances').mockResolvedValue({
      items: [
        {
          id: 'inst-1',
          name: 'Forge Client',
          mc_version: '1.21',
          loader: 'forge',
          created_at: 'now',
          updated_at: 'now',
        },
      ],
    });
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({
      items: [{ game_server_id: 'mon-1', instance_id: 'inst-1', instance_name: 'Forge Client' }],
    });
    vi.spyOn(api, 'myLauncherDevice').mockResolvedValue({
      device_id: 'dev-1',
      owner_type: 'user',
    });
    vi.spyOn(api, 'listProfiles').mockResolvedValue({
      items: [{ id: 'prof-1', username: 'Steve', offline_uuid: 'uuid', model: 'steve', created_at: 'now' }],
    });
    vi.spyOn(api, 'createLaunchRequest').mockResolvedValue({
      id: 'lr-1',
      status: 'queued',
      instance_id: 'inst-1',
      expires_at: new Date().toISOString(),
    });
    vi.spyOn(api, 'getLaunchRequest').mockResolvedValue({
      id: 'lr-1',
      status: 'completed',
      instance_id: 'inst-1',
      expires_at: new Date().toISOString(),
    });

    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await waitFor(() => expect(api.listMonitoringBindings).toHaveBeenCalled());
    await waitFor(() => expect(api.myLauncherDevice).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /Подключиться/ }));

    await waitFor(
      () =>
        expect(api.createLaunchRequest).toHaveBeenCalledWith({
          instance_id: 'inst-1',
          offline_profile_id: 'prof-1',
          join_server_address: 'play.example.com',
          join_server_port: 25565,
        }),
      { timeout: 5000 },
    );
  }, 20000);

  it('shows guest hint for QXLauncher binding on server cards', async () => {
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    expect(document.querySelector('.monitoring-connect-guest-hint')).toBeTruthy();
  });

  it('debounces search input before refetching servers', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
      renderWithProviders(<MonitoringPage />, '/monitoring');
      await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
      const callsBefore = vi.mocked(api.listMonitoringServers).mock.calls.length;
      const search = screen.getByPlaceholderText(/\u041f\u043e\u0438\u0441\u043a \u043f\u043e \u043d\u0430\u0437\u0432\u0430\u043d\u0438\u044e/i);
      await user.type(search, 'survival');
      await vi.advanceTimersByTimeAsync(399);
      expect(vi.mocked(api.listMonitoringServers).mock.calls.length).toBe(callsBefore);
      await vi.advanceTimersByTimeAsync(1);
      await waitFor(() =>
        expect(api.listMonitoringServers).toHaveBeenCalledWith(
          expect.objectContaining({ q: 'survival' }),
        ),
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it('sorts servers by name when sort option changes', async () => {
    vi.mocked(api.listMonitoringServers).mockResolvedValue({
      items: [
        { ...sampleServer, id: 'mon-2', name: 'Zeta Realm' },
        { ...sampleServer, id: 'mon-3', name: 'Alpha Realm' },
      ],
    });
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Zeta Realm')).toBeInTheDocument());

    const comboboxes = screen.getAllByRole('combobox');
    await user.click(comboboxes[4]);
    await user.click(await screen.findByText(/\u041f\u043e \u043d\u0430\u0437\u0432\u0430\u043d\u0438\u044e/i));

    const cards = screen.getAllByText(/Realm$/).map((el) => el.textContent);
    expect(cards.indexOf('Alpha Realm')).toBeLessThan(cards.indexOf('Zeta Realm'));
  });

  it('opens minecraft link when launch request fails without error toast', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
      saved_at: Date.now(),
    });
    vi.mocked(fetch).mockImplementation(mockAuthedFetch());
    vi.spyOn(api, 'listInstances').mockResolvedValue({
      items: [
        {
          id: 'inst-1',
          name: 'Forge Client',
          mc_version: '1.21',
          loader: 'forge',
          created_at: 'now',
          updated_at: 'now',
        },
      ],
    });
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({
      items: [{ game_server_id: 'mon-1', instance_id: 'inst-1', instance_name: 'Forge Client' }],
    });
    vi.spyOn(api, 'myLauncherDevice').mockResolvedValue({
      device_id: 'dev-1',
      owner_type: 'user',
    });
    vi.spyOn(api, 'listProfiles').mockResolvedValue({
      items: [{ id: 'prof-1', username: 'Steve', offline_uuid: 'uuid', model: 'steve', created_at: 'now' }],
    });
    vi.spyOn(api, 'createLaunchRequest').mockRejectedValue(new Error('launch failed'));

    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    await waitFor(() => expect(api.myLauncherDevice).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /\u041f\u043e\u0434\u043a\u043b\u044e\u0447\u0438\u0442\u044c\u0441\u044f/i }));

    await waitFor(() =>
      expect(testNavigation.hrefSet).toHaveBeenCalledWith('minecraft://play.example.com:25565'),
    );
    expect(testMessage.error).not.toHaveBeenCalled();
  });

  it('refreshes server list from hero button', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());
    vi.mocked(api.listMonitoringServers).mockClear();
    await user.click(screen.getByRole('button', { name: /\u041e\u0431\u043d\u043e\u0432\u0438\u0442\u044c/i }));
    await waitFor(() => expect(api.listMonitoringServers).toHaveBeenCalled());
  });

  it('rates a server when authenticated', async () => {
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
      saved_at: Date.now(),
    });
    vi.mocked(fetch).mockImplementation(mockAuthedFetch());
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<MonitoringPage />, '/monitoring');
    await waitFor(() => expect(screen.getByText('Survival World')).toBeInTheDocument());

    const card = screen.getByText('Survival World').closest('article');
    expect(card).not.toBeNull();
    const stars = within(card!).getAllByRole('radio');
    await user.click(stars[4]);
    await waitFor(() => expect(api.rateMonitoringServer).toHaveBeenCalledWith('mon-1', 5));
    expect(testMessage.success).toHaveBeenCalled();
  });

});
