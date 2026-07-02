import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from 'antd';
import { testMessage } from '@/test/test-message';
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
    vi.spyOn(vpsGameServers, 'restartVpsGameServer').mockResolvedValue({
      id: 'gs-1',
      name: 'Forge',
      server_type: 'forge',
      mc_version: '1.21',
      loader_version: '47.0.0',
      address: '127.0.0.1',
      port: 25565,
      status: 'starting',
      created_at: 'now',
    });
    vi.spyOn(Modal, 'confirm').mockImplementation(() => ({
      destroy: vi.fn(),
      update: vi.fn(),
    }));
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

  it('queues sync to selected game server and closes modal', async () => {
    const onClose = vi.fn();
    const user = userEvent.setup({ delay: null });
    renderWithTheme(
      <ModSyncModal open selection={selection} instanceLoader="forge" onClose={onClose} />,
    );

    await waitFor(() => expect(screen.getByText('Forge')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Синхронизировать' }));
    await waitFor(() => expect(api.syncModToGameServer).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(Modal.confirm).toHaveBeenCalled();
  });

  it('syncs installed required dependencies with the main mod', async () => {
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-1',
      version_number: '0.5.0',
      files: [{ filename: 'sodium-0.5.0.jar', url: 'https://example/mod.jar' }],
      dependencies: [
        {
          source: 'modrinth',
          project_id: 'dep-1',
          project_name: 'Fabric API',
          dependency_type: 'required',
          version_id: 'dep-ver',
          filename: 'fabric-api.jar',
          download_url: 'https://example/dep.jar',
        },
      ],
    });
    const onClose = vi.fn();
    const user = userEvent.setup({ delay: null });
    renderWithTheme(
      <ModSyncModal
        open
        selection={selection}
        instanceLoader="forge"
        installedResources={[
          {
            source: 'modrinth',
            project_id: 'dep-1',
            project_name: 'Fabric API',
            version_id: 'dep-ver',
            filename: 'fabric-api.jar',
            resource_type: 'mod',
            installed_at: 'now',
          },
        ]}
        onClose={onClose}
      />,
    );

    await waitFor(() => expect(screen.getByText('Forge')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Синхронизировать' }));
    await waitFor(() => expect(api.syncModToGameServer).toHaveBeenCalledTimes(2));
    expect(api.syncModToGameServer).toHaveBeenNthCalledWith(
      1,
      'vps-1',
      'gs-1',
      expect.objectContaining({ project_id: 'dep-1' }),
    );
    expect(api.syncModToGameServer).toHaveBeenNthCalledWith(
      2,
      'vps-1',
      'gs-1',
      expect.objectContaining({ project_id: 'proj-1' }),
    );
  });

  it('restarts server when user confirms after sync', async () => {
    vi.mocked(Modal.confirm).mockImplementation((config) => {
      void config.onOk?.();
      return { destroy: vi.fn(), update: vi.fn() };
    });
    const user = userEvent.setup({ delay: null });
    renderWithTheme(
      <ModSyncModal open selection={selection} instanceLoader="forge" onClose={vi.fn()} />,
    );

    await waitFor(() => expect(screen.getByText('Forge')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Синхронизировать' }));
    await waitFor(() => expect(vpsGameServers.restartVpsGameServer).toHaveBeenCalledWith('vps-1', 'gs-1'));
    expect(testMessage.success).toHaveBeenCalledWith('Игровой сервер перезапускается…');
  });

  it('does not restart when user declines after sync', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(
      <ModSyncModal open selection={selection} instanceLoader="forge" onClose={vi.fn()} />,
    );

    await waitFor(() => expect(screen.getByText('Forge')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Синхронизировать' }));
    await waitFor(() => expect(Modal.confirm).toHaveBeenCalled());
    expect(vpsGameServers.restartVpsGameServer).not.toHaveBeenCalled();
  });
});
