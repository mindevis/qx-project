import { describe, expect, it, vi, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerVersionSettingsPanel } from './GameServerVersionSettingsPanel';
import type { VpsGameServer } from '@/lib/vpsGameServers';
import * as gameServerVersions from '@/lib/gameServerVersions';
import * as mcVersionsCache from '@/lib/mcVersionsCache';

const game: VpsGameServer = {
  id: 'gs-1',
  name: 'Paper',
  server_type: 'paper',
  mc_version: '1.21',
  loader_version: '10',
  status: 'stopped',
  created_at: 'now',
};

describe('GameServerVersionSettingsPanel', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('applies a new Minecraft and core version', async () => {
    const user = userEvent.setup({ delay: null });
    const onUpdated = vi.fn();
    vi.spyOn(mcVersionsCache, 'cachedListMcVersions').mockResolvedValue({
      latest: { release: '1.21.1' },
      items: [
        { id: '1.21.1', type: 'release' },
        { id: '1.21', type: 'release' },
      ],
    });
    vi.spyOn(gameServerVersions, 'listGameServerMcVersions').mockResolvedValue([
      { value: '1.21.1', label: '1.21.1' },
      { value: '1.21', label: '1.21' },
    ]);
    vi.spyOn(gameServerVersions, 'listGameServerLoaderVersions').mockImplementation(
      async (_type, mc) =>
        mc === '1.21.1'
          ? [{ value: '20', label: '20' }]
          : [{ value: '10', label: '10' }],
    );
    const change = vi.spyOn(api, 'changeVpsGameServerVersion').mockResolvedValue({
      ...game,
      mc_version: '1.21.1',
      loader_version: '20',
      status: 'installing',
    });

    renderWithTheme(
      <GameServerVersionSettingsPanel vpsId="srv-1" game={game} onUpdated={onUpdated} />,
    );

    await waitFor(() => expect(screen.getByText('Версия Minecraft и ядра')).toBeInTheDocument());
    await waitFor(() => expect(screen.getByRole('button', { name: 'Сменить версию' })).toBeDisabled());

    const comboboxes = screen.getAllByRole('combobox');
    await user.click(comboboxes[0]!);
    await user.click(await screen.findByTitle('1.21.1'));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Сменить версию' })).not.toBeDisabled(),
    );
    await user.click(screen.getByRole('button', { name: 'Сменить версию' }));

    await waitFor(() => expect(change).toHaveBeenCalled());
    expect(change).toHaveBeenCalledWith('srv-1', 'gs-1', {
      mc_version: '1.21.1',
      loader_version: '20',
    });
    expect(onUpdated).toHaveBeenCalled();
    expect(testMessage.success).toHaveBeenCalled();
  });
});
