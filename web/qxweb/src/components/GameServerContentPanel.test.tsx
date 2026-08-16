import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { clearModCatalogCaches } from '@/lib/modCatalogCache';
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
    window.localStorage.clear();
    clearModCatalogCaches();
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

  it('does not treat stale catalog records as installed after files are gone', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null, pointerEventsCheck: 0 });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listGameServerResources').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          project_id: 'sodium',
          project_name: 'Sodium',
          filename: 'sodium-fabric-0.5.8.jar',
          resource_type: 'mod',
          installed_at: 'now',
        },
      ],
    });
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

    await waitFor(() => expect(screen.getByText('Папка модов пуста')).toBeInTheDocument());
    await user.click(screen.getByRole('radio', { name: /Каталог/ }));
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(screen.queryByText('Установлен')).not.toBeInTheDocument();
  });

  it('shows catalog metadata on installed mods', async () => {
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({
      items: [{ name: 'jei.jar', path: 'mods/jei.jar', dir: false, size: 2048 }],
    });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listGameServerResources').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          project_id: 'jei',
          project_name: 'Just Enough Items',
          version_number: '19.8.0',
          filename: 'jei.jar',
          resource_type: 'mod',
          icon_url: 'https://example.com/jei.png',
          downloads: 12000,
          installed_at: 'now',
          side_override: 'server',
        },
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

    await waitFor(() => expect(screen.getByText('Just Enough Items')).toBeInTheDocument());
    expect(screen.getByText('jei.jar')).toBeInTheDocument();
    expect(screen.getByText('19.8.0')).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Тип мода (сторона)' })).toBeInTheDocument();
  });

  it('lists server and client mods together and can change the side', async () => {
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
    const patch = vi.spyOn(api, 'patchGameServerResource').mockResolvedValue({ status: 'ok' });

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
    expect(screen.getByText('journeymap.jar')).toBeInTheDocument();

    const sideSelects = screen.getAllByRole('combobox', { name: 'Тип мода (сторона)' });
    await user.click(sideSelects[0]);
    await user.click(await screen.findByTitle('Сервер'));

    await waitFor(() =>
      expect(patch).toHaveBeenCalledWith(
        'srv-1',
        'gs-1',
        expect.objectContaining({ filename: 'bettercombat.jar', side_override: 'server' }),
      ),
    );
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
      mod_target: 'mods',
      side_override: 'both',
    });
  });

  it('installs a client-only catalog mod into client-mods', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'sodium',
          slug: 'sodium',
          name: 'Sodium',
          summary: 'Perf',
          project_type: 'mod',
          client_side: 'required',
          server_side: 'unsupported',
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
          files: [{ filename: 'sodium.jar', url: 'https://cdn.modrinth.com/sodium.jar', size: 10 }],
        },
      ],
    });
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-1',
      version_number: '1.0.0',
      game_versions: ['1.21'],
      loaders: ['forge'],
      files: [{ filename: 'sodium.jar', url: 'https://cdn.modrinth.com/sodium.jar', size: 10 }],
      dependencies: [],
    });
    const sync = vi.spyOn(api, 'syncModToGameServer').mockResolvedValue({
      status: 'installed',
      filename: 'sodium.jar',
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
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Установить' }));
    await waitFor(() => expect(sync).toHaveBeenCalled());
    expect(sync.mock.calls[0][2]).toMatchObject({
      filename: 'sodium.jar',
      mod_target: 'client-mods',
      side_override: 'client',
    });
  });

  it('switches the catalog to a table view', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null, pointerEventsCheck: 0 });
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

    await user.click(screen.getByRole('radio', { name: 'Таблица' }));
    await user.click(screen.getByText('Каталог'));
    await waitFor(() => expect(screen.getByText('JEI')).toBeInTheDocument());
    expect(document.querySelector('.qxmods-catalog-table')).toBeTruthy();
  });

  it('merges same-name catalog mods from both sources into one card', async () => {
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
          downloads: 200,
          external_url: 'https://modrinth.com/mod/jei',
        },
        {
          source: 'curseforge',
          id: '238222',
          slug: 'jei',
          name: 'JEI',
          summary: 'Just Enough Items',
          project_type: 'mod',
          downloads: 80,
          external_url: 'https://www.curseforge.com/minecraft/mc-mods/jei',
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

    await user.click(screen.getByText('Каталог'));
    await waitFor(() => expect(screen.getByRole('button', { name: 'JEI' })).toBeInTheDocument());
    expect(screen.getAllByRole('button', { name: 'JEI' })).toHaveLength(1);
    expect(screen.getByRole('radio', { name: 'Modrinth' })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'CurseForge' })).toBeInTheDocument();
  });

  it('opens the dependency wizard on a game server without InstanceModsProvider', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'better-combat',
          slug: 'better-combat',
          name: 'Better Combat',
          summary: 'Combat',
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
          id: 'ver-bc',
          version_number: '2.0.0',
          game_versions: ['1.21'],
          loaders: ['forge'],
          files: [{ filename: 'bettercombat.jar', url: 'https://cdn.modrinth.com/bc.jar', size: 10 }],
        },
      ],
    });
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-bc',
      version_number: '2.0.0',
      game_versions: ['1.21'],
      loaders: ['forge'],
      files: [{ filename: 'bettercombat.jar', url: 'https://cdn.modrinth.com/bc.jar', size: 10 }],
      dependencies: [
        {
          source: 'modrinth',
          project_id: 'cloth-config',
          project_name: 'Cloth Config API',
          dependency_type: 'required',
          version_id: 'ver-cloth',
          filename: 'cloth.jar',
          download_url: 'https://cdn.modrinth.com/cloth.jar',
        },
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

    await user.click(screen.getByText('Каталог'));
    await waitFor(() => expect(screen.getByText('Better Combat')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Установить' }));
    await waitFor(() => expect(screen.getByText('Обязательные зависимости')).toBeInTheDocument());
    expect(screen.getByText('Cloth Config API')).toBeInTheDocument();
  });

  it('lists installed resource packs and browses the resource pack catalog', async () => {
    vi.spyOn(api, 'listVpsGameServerResourcepacks').mockResolvedValue({
      items: [{ name: 'faithful.zip', path: 'resourcepacks/faithful.zip', dir: false, size: 2048 }],
    });
    vi.spyOn(api, 'listVpsGameServerClientResourcepacks').mockResolvedValue({ items: [] });
    const browse = vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'faithful',
          slug: 'faithful',
          name: 'Faithful 32x',
          project_type: 'resourcepack',
          external_url: '',
        },
      ],
      has_more: false,
      curseforge_enabled: false,
    });

    renderWithTheme(
      <GameServerContentPanel
        kind="resourcepack"
        vpsId="srv-1"
        gameServerId="gs-1"
        agentOnline={true}
        supported={true}
        serverType="forge"
        mcVersion="1.21"
      />,
    );

    await waitFor(() => expect(screen.getByText('faithful.zip')).toBeInTheDocument());
    await waitFor(() =>
      expect(browse).toHaveBeenCalledWith(expect.objectContaining({ type: 'resourcepack' })),
    );
  });
});
