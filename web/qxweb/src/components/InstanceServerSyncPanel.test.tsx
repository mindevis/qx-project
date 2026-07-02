import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { renderWithProviders } from '@/test/test-utils';
import * as gameServerSyncTargets from '@/lib/gameServerSyncTargets';
import {
  InstanceResourceSyncButton,
  InstanceServerSyncProvider,
} from './InstanceServerSyncPanel';

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
    name: 'qRPG TechnoMagic',
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

function renderSyncButton(
  items = [installedMod],
  canSync = true,
  targets = [syncTarget],
) {
  vi.spyOn(gameServerSyncTargets, 'loadGameServerSyncTargets').mockResolvedValue(targets);
  return renderWithProviders(
    <InstanceModsProvider instance={forgeInstance} canSync={canSync}>
      <InstanceServerSyncProvider items={items}>
        <InstanceResourceSyncButton item={items[0]} />
      </InstanceServerSyncProvider>
    </InstanceModsProvider>,
  );
}

describe('InstanceResourceSyncButton', () => {
  beforeEach(() => {
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-1',
      version_number: '5.10.3',
      files: [{ filename: installedMod.filename, url: 'https://example/mod.jar' }],
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows sync button when mod is not on any server', async () => {
    renderSyncButton();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Синхронизировать с сервером/ })).toBeInTheDocument(),
    );
  });

  it('opens server picker modal when sync button is clicked', async () => {
    const user = userEvent.setup({ delay: null });
    renderSyncButton();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Синхронизировать с сервером/ })).toBeEnabled(),
    );

    await user.click(screen.getByRole('button', { name: /Синхронизировать с сервером/ }));

    await waitFor(() => expect(screen.getByText('Синхронизация с сервером')).toBeInTheDocument());
    expect(screen.getByText('qRPG TechnoMagic')).toBeInTheDocument();
  });

  it('shows checkmark when mod is already on a server', async () => {
    renderSyncButton([installedMod], true, [
      {
        ...syncTarget,
        serverMods: [{ name: installedMod.filename, path: 'mods/' + installedMod.filename, dir: false }],
      },
    ]);

    await waitFor(() =>
      expect(screen.getByLabelText(/Синхронизирован с «qRPG TechnoMagic»/)).toBeInTheDocument(),
    );
    expect(screen.queryByRole('button', { name: /Синхронизировать с сервером/ })).not.toBeInTheDocument();
  });

  it('detects synced mods by catalog version filename', async () => {
    const catalogFilename = 'journeymap-1.20.1-5.10.3-forge.jar';
    vi.spyOn(api, 'getModVersion').mockResolvedValue({
      id: 'ver-1',
      version_number: '5.10.3',
      files: [{ filename: catalogFilename, url: 'https://example/mod.jar' }],
    });

    renderSyncButton(
      [{ ...installedMod, filename: 'different-local-name.jar' }],
      true,
      [
        {
          ...syncTarget,
          serverMods: [{ name: catalogFilename, path: `mods/${catalogFilename}`, dir: false }],
        },
      ],
    );

    await waitFor(() =>
      expect(screen.getByLabelText(/Синхронизирован с «qRPG TechnoMagic»/)).toBeInTheDocument(),
    );
  });

  it('prompts sign-in for guests', async () => {
    renderSyncButton([installedMod], false);
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Синхронизировать с сервером/ })).toBeInTheDocument(),
    );
  });
});
