import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import * as vpsGameServers from '@/lib/vpsGameServers';
import { ModSyncModal, type ModSyncSelection } from './ModSyncModal';

const selection: ModSyncSelection = {
  source: 'modrinth',
  projectId: 'proj-1',
  projectName: 'Sodium',
  version: {
    id: 'ver-1',
    version_number: '0.5.0',
    game_versions: ['1.21'],
    files: [{ filename: 'sodium-0.5.0.jar', url: 'https://example/mod.jar' }],
  },
};

describe('ModSyncModal', () => {
  beforeEach(() => {
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'info').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'listServers').mockResolvedValue({
      items: [
        {
          id: 'vps-1',
          name: 'Survival VPS',
          slug: 'survival',
          status: 'running',
          agent_online: true,
        },
      ],
    });
    vi.spyOn(vpsGameServers, 'listVpsGameServers').mockResolvedValue([
      {
        id: 'gs-1',
        name: 'Forge',
        server_type: 'forge',
        mc_version: '1.21',
        loader_version: '47.0.0',
        address: '127.0.0.1',
        port: 25565,
        status: 'running',
        created_at: 'now',
      },
    ]);
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'syncModToGameServer').mockResolvedValue({ status: 'queued' });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows empty state when no sync targets exist', async () => {
    vi.mocked(vpsGameServers.listVpsGameServers).mockResolvedValueOnce([]);
    renderWithTheme(
      <ModSyncModal open selection={selection} instanceLoader="forge" onClose={vi.fn()} />,
    );
    await waitFor(() =>
      expect(screen.getByText('Нет доступных серверов с агентом и поддержкой модов')).toBeInTheDocument(),
    );
  });

  it('queues sync to selected game server', async () => {
    const onClose = vi.fn();
    const user = userEvent.setup({ delay: null });
    renderWithTheme(
      <ModSyncModal open selection={selection} instanceLoader="forge" onClose={onClose} />,
    );

    await waitFor(() => expect(screen.getByText('Forge')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Синхронизировать' }));
    await waitFor(() => expect(api.syncModToGameServer).toHaveBeenCalled());
    expect(message.success).toHaveBeenCalledWith('Синхронизация поставлена в очередь');
    expect(onClose).toHaveBeenCalled();
  });
});
