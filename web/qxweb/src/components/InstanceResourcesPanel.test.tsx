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

const sodiumItem = {
  source: 'modrinth' as const,
  id: 'sodium',
  name: 'Sodium',
  summary: 'Performance mod',
  external_url: 'https://modrinth.com/mod/sodium',
  client_side: 'unsupported',
  server_side: 'required',
};

describe('InstanceResourcesPanel', () => {
  beforeEach(() => {
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [sodiumItem],
      has_more: false,
      curseforge_enabled: true,
    });
    vi.spyOn(api, 'searchMods').mockResolvedValue({
      items: [sodiumItem],
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

  it('loads catalog on mount and opens detail drawer', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(api.browseMods).toHaveBeenCalledWith({
      type: 'mod',
      loader: 'forge',
      mc_version: '1.21',
      source: 'all',
      sort: 'downloads',
      limit: 20,
      offset: 0,
    });

    await user.click(screen.getByRole('button', { name: 'Открыть' }));
    await waitFor(() => expect(screen.getByText('0.5.0')).toBeInTheDocument());
    expect(api.listModVersions).toHaveBeenCalled();
  });

  it('shows curseforge disabled notice when api key missing', async () => {
    vi.mocked(api.browseMods).mockResolvedValueOnce({
      items: [],
      has_more: false,
      curseforge_enabled: false,
    });
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await waitFor(() =>
      expect(screen.getByText(/CurseForge отключён/)).toBeInTheDocument(),
    );
  });

  it('shows instance loader and mc version filter context', async () => {
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);
    await waitFor(() =>
      expect(
        screen.getByText('Каталог отфильтрован для Minecraft 1.21 · forge'),
      ).toBeInTheDocument(),
    );
  });

  it('uses search api when optional name filter is applied', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await waitFor(() => expect(api.browseMods).toHaveBeenCalled());

    await user.type(screen.getByPlaceholderText('Необязательно: сузить по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() =>
      expect(api.searchMods).toHaveBeenCalledWith({
        q: 'sodium',
        type: 'mod',
        loader: 'forge',
        mc_version: '1.21',
        limit: 20,
      }),
    );
  });

  it('shows browse error from api', async () => {
    vi.mocked(api.browseMods).mockRejectedValueOnce(new Error('browse failed'));
    renderWithTheme(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('browse failed'));
  });
});
