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
});
