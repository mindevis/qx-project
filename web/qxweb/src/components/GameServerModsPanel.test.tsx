import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerModsPanel } from './GameServerModsPanel';

describe('GameServerModsPanel', () => {
  beforeEach(() => {
    vi.spyOn(api, 'listGameServerResources').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listModVersions').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [],
      has_more: false,
      curseforge_enabled: false,
    });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerResourcepacks').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientResourcepacks').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerShaders').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientShaders').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerDatapacks').mockResolvedValue({ items: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows unsupported message for vanilla', () => {
    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supportsMods={false}
      />,
    );
    expect(screen.getByText(/не поддерживает моды/i)).toBeInTheDocument();
  });

  it('shows agent required when offline', () => {
    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={false}
        supportsMods={true}
      />,
    );
    expect(screen.getByText(/Deploy/i)).toBeInTheDocument();
  });

  it('lists mod jars and handles errors', async () => {
    vi.spyOn(api, 'listVpsGameServerMods')
      .mockResolvedValueOnce({
        items: [
          { name: 'mods', path: 'mods', dir: true },
          { name: 'example.jar', path: 'mods/example.jar', dir: false, size: 2048 },
          { name: 'huge.jar', path: 'mods/huge.jar', dir: false, size: 3 * 1024 * 1024 },
        ],
      })
      .mockRejectedValueOnce(new Error('mods failed'));
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [],
      has_more: false,
      curseforge_enabled: false,
    });

    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supportsMods={true}
      />,
    );

    await waitFor(() => expect(screen.getByText('example.jar')).toBeInTheDocument());
    expect(screen.getByText('2.0 KB')).toBeInTheDocument();
    expect(screen.getByText('3.0 MB')).toBeInTheDocument();

    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-2"
        agentOnline={true}
        supportsMods={true}
      />,
    );
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('mods failed'));
  });

  it('shows empty mods list', async () => {
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [],
      has_more: false,
      curseforge_enabled: false,
    });

    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supportsMods={true}
      />,
    );
    await waitFor(() => expect(screen.getByText('Папка модов пуста')).toBeInTheDocument());
  });

  it('formats unknown file sizes', async () => {
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({
      items: [{ name: 'empty.jar', path: 'mods/empty.jar', dir: false }],
    });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [],
      has_more: false,
      curseforge_enabled: false,
    });

    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supportsMods={true}
      />,
    );
    await waitFor(() => expect(screen.getByText('—')).toBeInTheDocument());
  });

  it('switches the catalog to resource packs, shaders, and datapacks', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null });
    const browse = vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [],
      has_more: false,
      curseforge_enabled: false,
    });

    renderWithTheme(
      <GameServerModsPanel
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supportsMods={true}
        serverType="forge"
      />,
    );

    await waitFor(() => expect(screen.getByText('Ресурспаки')).toBeInTheDocument());
    expect(screen.getByText('Шейдеры')).toBeInTheDocument();
    expect(screen.getByText('Датапаки')).toBeInTheDocument();

    await user.click(screen.getByText('Ресурспаки'));
    await waitFor(() =>
      expect(browse).toHaveBeenCalledWith(expect.objectContaining({ type: 'resourcepack' })),
    );

    await user.click(screen.getByText('Шейдеры'));
    await waitFor(() =>
      expect(browse).toHaveBeenCalledWith(expect.objectContaining({ type: 'shader' })),
    );

    await user.click(screen.getByText('Датапаки'));
    await waitFor(() =>
      expect(browse).toHaveBeenCalledWith(expect.objectContaining({ type: 'datapack' })),
    );
  });
});
