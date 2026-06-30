import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { InstanceResourcesPanel } from './InstanceResourcesPanel';

const forgeInstance = {
  id: 'inst-1',
  name: 'Forge',
  mc_version: '1.21',
  loader: 'forge',
  loader_version: '47.0.0',
  created_at: 'now',
  updated_at: 'now',
};

describe('InstanceResourcesPanel', () => {
  beforeEach(() => {
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'searchMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'sodium',
          name: 'Sodium',
          summary: 'Performance mod',
          external_url: 'https://modrinth.com/mod/sodium',
          client_side: 'unsupported',
          server_side: 'required',
        },
      ],
      curseforge_enabled: true,
    });
    vi.spyOn(api, 'listModVersions').mockResolvedValue({
      items: [
        {
          id: 'ver-1',
          version_number: '0.5.0',
          game_versions: ['1.21'],
          files: [{ filename: 'sodium.jar', url: 'https://example/sodium.jar' }],
        },
      ],
    });
    vi.spyOn(api, 'listServers').mockResolvedValue({ items: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns null for vanilla instances', () => {
    renderWithTheme(
      <InstanceResourcesPanel
        instance={{ ...forgeInstance, loader: 'vanilla' }}
        canSync={false}
      />,
    );
    expect(screen.queryByLabelText('Ресурсы')).not.toBeInTheDocument();
  });

  it('searches mods and opens detail drawer', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await user.type(screen.getByPlaceholderText('Поиск по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Открыть' }));
    await waitFor(() => expect(screen.getByText('0.5.0')).toBeInTheDocument());
    expect(api.listModVersions).toHaveBeenCalled();
  });

  it('shows curseforge disabled notice when api key missing', async () => {
    vi.mocked(api.searchMods).mockResolvedValueOnce({
      items: [],
      curseforge_enabled: false,
    });
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await user.type(screen.getByPlaceholderText('Поиск по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() =>
      expect(screen.getByText(/CurseForge отключён/)).toBeInTheDocument(),
    );
    expect(api.searchMods).toHaveBeenCalledWith({
      q: 'sodium',
      type: 'mod',
      loader: 'forge',
      mc_version: '1.21',
    });
  });

  it('shows instance loader and mc version filter context', () => {
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);
    expect(
      screen.getByText('Результаты отфильтрованы для Minecraft 1.21 · forge'),
    ).toBeInTheDocument();
  });

  it('shows search error from api', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(api.searchMods).mockRejectedValueOnce(new Error('search failed'));
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await user.type(screen.getByPlaceholderText('Поиск по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('search failed'));
  });
});
