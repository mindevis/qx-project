import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from 'antd';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { renderWithProviders } from '@/test/test-utils';
import * as gameServerSyncTargets from '@/lib/gameServerSyncTargets';
import { InstanceServerSyncPanel } from './InstanceServerSyncPanel';

const forgeInstance = {
  id: 'inst-1',
  name: 'Forge',
  mc_version: '1.20.1',
  loader: 'forge',
  loader_version: '47.1.20',
  created_at: 'now',
  updated_at: 'now',
};

const installedMod = {
  source: 'curseforge' as const,
  project_id: 'journeymap',
  project_name: 'JourneyMap',
  version_id: 'ver-1',
  version_number: '5.10.3',
  filename: 'journeymap-1.20.1-5.10.3-forge.jar',
  resource_type: 'mod' as const,
  installed_at: '2026-01-01T00:00:00Z',
};

const syncTarget = {
  vpsId: 'vps-1',
  vpsName: 'My VPS',
  gameServer: {
    id: 'gs-1',
    name: 'qRPG',
    server_type: 'forge' as const,
    mc_version: '1.20.1',
    loader_version: '47.1.20',
    address: '127.0.0.1',
    port: 25565,
    status: 'running' as const,
    created_at: 'now',
  },
  serverMods: [] as { name: string; path: string; dir: boolean }[],
};

function renderPanel(items = [installedMod], canSync = true) {
  return renderWithProviders(
    <InstanceModsProvider instance={forgeInstance} canSync={canSync}>
      <InstanceServerSyncPanel items={items} />
    </InstanceModsProvider>,
  );
}

describe('InstanceServerSyncPanel', () => {
  beforeEach(() => {
    vi.spyOn(gameServerSyncTargets, 'loadGameServerSyncTargets').mockResolvedValue([syncTarget]);
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-1',
      version_number: '5.10.3',
      files: [{ filename: installedMod.filename, url: 'https://example/mod.jar' }],
    });
    vi.spyOn(api, 'syncModToGameServer').mockResolvedValue({ status: 'queued' });
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(Modal, 'confirm').mockImplementation(() => ({
      destroy: vi.fn(),
      update: vi.fn(),
    }));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows sign-in hint for guests', () => {
    renderPanel([installedMod], false);
    expect(screen.getByText('Войдите, чтобы синхронизировать моды с сервером.')).toBeInTheDocument();
  });

  it('shows server selector, status and sync button', async () => {
    renderPanel();
    await waitFor(() => expect(screen.getByText('Синхронизация с сервером')).toBeInTheDocument());
    expect(screen.getByRole('combobox')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Синхронизировать/ })).toBeInTheDocument();
    expect(screen.getByText('Ожидают: 1 из 1')).toBeInTheDocument();
  });

  it('queues missing mods to selected server', async () => {
    const user = userEvent.setup({ delay: null });
    renderPanel();
    await waitFor(() => expect(screen.getByRole('button', { name: /Синхронизировать/ })).toBeEnabled());

    await user.click(screen.getByRole('button', { name: /Синхронизировать/ }));

    await waitFor(() =>
      expect(api.syncModToGameServer).toHaveBeenCalledWith('vps-1', 'gs-1', {
        source: 'curseforge',
        project_id: 'journeymap',
        version_id: 'ver-1',
        filename: installedMod.filename,
        download_url: 'https://example/mod.jar',
        project_name: 'JourneyMap',
        version_number: '5.10.3',
      }),
    );
    expect(testMessage.success).toHaveBeenCalledWith('В очередь отправлено модов: 1');
  });
});
