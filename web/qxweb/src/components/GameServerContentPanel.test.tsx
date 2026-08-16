import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerContentPanel } from './GameServerContentPanel';

function mockCatalogApis() {
  vi.spyOn(api, 'browseMods').mockResolvedValue({
    items: [],
    has_more: false,
    curseforge_enabled: false,
  });
  vi.spyOn(api, 'searchMods').mockResolvedValue({
    items: [],
    curseforge_enabled: false,
  });
  vi.spyOn(api, 'listGameServerResources').mockResolvedValue({ items: [] });
  vi.spyOn(api, 'listModVersions').mockResolvedValue({ items: [] });
}

describe('GameServerContentPanel', () => {
  beforeEach(() => {
    mockCatalogApis();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows unsupported message for plugins on forge', () => {
    renderWithTheme(
      <GameServerContentPanel
        kind="plugin"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={false}
        serverType="forge"
        mcVersion="1.21"
      />,
    );
    expect(screen.getByText(/не поддерживает плагины/i)).toBeInTheDocument();
  });

  it('lists installed plugins', async () => {
    vi.spyOn(api, 'listVpsGameServerPlugins').mockResolvedValue({
      items: [{ name: 'EssentialsX.jar', path: 'plugins/EssentialsX.jar', dir: false, size: 1024 }],
    });

    renderWithTheme(
      <GameServerContentPanel
        kind="plugin"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={true}
        serverType="paper"
        mcVersion="1.21"
      />,
    );

    await waitFor(() => expect(screen.getByText('EssentialsX.jar')).toBeInTheDocument());
  });

  it('lists installed datapacks and handles errors', async () => {
    vi.spyOn(api, 'listVpsGameServerDatapacks')
      .mockResolvedValueOnce({
        items: [{ name: 'vanilla-tweaks.zip', path: 'world/datapacks/vanilla-tweaks.zip', dir: false }],
      })
      .mockRejectedValueOnce(new Error('datapacks failed'));

    renderWithTheme(
      <GameServerContentPanel
        kind="datapack"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={true}
        serverType="vanilla"
        mcVersion="1.21"
      />,
    );

    await waitFor(() => expect(screen.getByText('vanilla-tweaks.zip')).toBeInTheDocument());

    renderWithTheme(
      <GameServerContentPanel
        kind="datapack"
        vpsId="srv-1"
        gameServerId="gs-2"
        agentOnline={true}
        supported={true}
        serverType="vanilla"
        mcVersion="1.21"
      />,
    );
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('datapacks failed'));
  });

  it('hides installed catalog items by default and can show only them', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({
      items: [{ name: 'sodium-fabric-0.5.8.jar', path: 'mods/sodium-fabric-0.5.8.jar', dir: false, size: 10 }],
    });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'sodium',
          slug: 'sodium',
          name: 'Sodium',
          summary: 'Performance',
          project_type: 'mod',
          external_url: '',
        },
        {
          source: 'modrinth',
          id: 'jei',
          slug: 'jei',
          name: 'JEI',
          summary: 'Items',
          project_type: 'mod',
          external_url: '',
        },
      ],
      has_more: false,
      curseforge_enabled: true,
    });

    renderWithTheme(
      <GameServerContentPanel
        kind="mod"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={true}
        serverType="forge"
        mcVersion="1.21"
      />,
    );

    await waitFor(() => expect(screen.getByText('sodium-fabric-0.5.8.jar')).toBeInTheDocument());
    await user.click(screen.getByText('Каталог'));
    await waitFor(() => expect(screen.getByText('JEI')).toBeInTheDocument());
    expect(screen.queryByText('Sodium')).not.toBeInTheDocument();

    await user.click(screen.getByRole('switch', { name: 'Только установленные' }));
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(screen.queryByText('JEI')).not.toBeInTheDocument();
  });

  it('filters installed mods by server and client folder', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({
      items: [
        { name: 'bettercombat.jar', path: 'mods/bettercombat.jar', dir: false, size: 10 },
      ],
    });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({
      items: [
        { name: 'journeymap.jar', path: 'client-mods/journeymap.jar', dir: false, size: 8 },
      ],
    });

    renderWithTheme(
      <GameServerContentPanel
        kind="mod"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={true}
        serverType="forge"
        mcVersion="1.21"
      />,
    );

    await waitFor(() => expect(screen.getByText('bettercombat.jar')).toBeInTheDocument());
    expect(screen.queryByText('journeymap.jar')).not.toBeInTheDocument();

    await user.click(screen.getByText('Клиентские моды'));
    await waitFor(() => expect(screen.getByText('journeymap.jar')).toBeInTheDocument());
    expect(screen.queryByText('bettercombat.jar')).not.toBeInTheDocument();
  });

  it('installs a catalog mod onto the game server', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'jei',
          slug: 'jei',
          name: 'JEI',
          summary: 'Items',
          project_type: 'mod',
          external_url: '',
        },
      ],
      has_more: false,
      curseforge_enabled: true,
    });
    vi.spyOn(api, 'listModVersions').mockResolvedValue({
      items: [
        {
          id: 'ver-1',
          version_number: '1.0.0',
          game_versions: ['1.21'],
          loaders: ['forge'],
          files: [{ filename: 'jei.jar', url: 'https://cdn.modrinth.com/jei.jar', size: 10 }],
        },
      ],
    });
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-1',
      version_number: '1.0.0',
      game_versions: ['1.21'],
      loaders: ['forge'],
      files: [{ filename: 'jei.jar', url: 'https://cdn.modrinth.com/jei.jar', size: 10 }],
      dependencies: [],
    });
    const sync = vi.spyOn(api, 'syncModToGameServer').mockResolvedValue({
      status: 'installed',
      filename: 'jei.jar',
    });

    renderWithTheme(
      <GameServerContentPanel
        kind="mod"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={true}
        serverType="forge"
        mcVersion="1.21"
      />,
    );

    await user.click(screen.getByText('Каталог'));
    await waitFor(() => expect(screen.getByText('JEI')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Установить' }));
    await waitFor(() => expect(sync).toHaveBeenCalled());
    expect(sync.mock.calls[0][2]).toMatchObject({
      source: 'modrinth',
      project_id: 'jei',
      filename: 'jei.jar',
    });
  });
});
