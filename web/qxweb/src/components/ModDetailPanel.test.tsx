import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { ModDetailPanel } from '@/components/ModDetailPanel';
import { clearModVersionCache } from '@/components/ModCatalogInstallControls';
import { I18nProvider } from '@/i18n/I18nContext';
import { ThemeProvider } from '@/theme/ThemeContext';
import { render } from '@testing-library/react';
import * as vpsGameServers from '@/lib/vpsGameServers';

const forgeInstance = {
  id: 'inst-1',
  name: 'Forge',
  mc_version: '1.21',
  loader: 'forge',
  loader_version: '47.0.0',
  created_at: 'now',
  updated_at: 'now',
};

const project = {
  source: 'modrinth' as const,
  id: 'sodium',
  name: 'Sodium',
  summary: 'Performance mod',
  external_url: 'https://modrinth.com/mod/sodium',
  client_side: 'required',
  server_side: 'required',
  project_type: 'mod' as const,
  game_versions: ['1.21'],
  loaders: ['forge'],
};

const modVersion = {
  id: 'ver-1',
  version_number: '0.5.0',
  game_versions: ['1.21'],
  loaders: ['forge'],
  files: [{ filename: 'sodium.jar', url: 'https://example.com/sodium.jar', primary: true }],
};

function renderDetail(canSync = true) {
  return render(
    <I18nProvider>
      <ThemeProvider>
        <MemoryRouter initialEntries={['/catalog/modrinth/sodium']}>
          <Routes>
            <Route
              path="/catalog/:source/:projectId"
              element={
                <InstanceModsProvider instance={forgeInstance} canSync={canSync}>
                  <ModDetailPanel />
                </InstanceModsProvider>
              }
            />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </I18nProvider>,
  );
}

describe('ModDetailPanel', () => {
  beforeEach(() => {
    clearModVersionCache();
    vi.spyOn(api, 'getModProject').mockResolvedValue(project);
    vi.spyOn(api, 'listModVersions').mockResolvedValue({ items: [modVersion] });
    vi.spyOn(api, 'getModVersion').mockResolvedValue({ ...modVersion, dependencies: [] });
    vi.spyOn(api, 'listInstanceResources').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'createModInstallRequest').mockResolvedValue({
      id: 'req-1',
      instance_id: 'inst-1',
      status: 'queued',
      created_at: 'now',
      updated_at: 'now',
    });
    vi.spyOn(api, 'getModInstallRequest').mockResolvedValue({
      id: 'req-1',
      instance_id: 'inst-1',
      status: 'completed',
      created_at: 'now',
      updated_at: 'now',
    });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('loads project detail and version list', async () => {
    renderDetail();
    await waitFor(() => expect(screen.getByRole('heading', { name: /Sodium/ })).toBeInTheDocument());
    expect(screen.getByText('Performance mod')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Установить' })).toBeInTheDocument();
    expect(api.getModProject).toHaveBeenCalledWith('modrinth', 'sodium');
    expect(api.listModVersions).toHaveBeenCalled();
  });

  it('installs selected version and opens server sync modal', async () => {
    vi.spyOn(api, 'listServers').mockResolvedValue({
      items: [{ id: 'vps-1', name: 'VPS', slug: 'vps', status: 'running', agent_online: true }],
    });
    vi.spyOn(vpsGameServers, 'listVpsGameServers').mockResolvedValue([]);
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    const user = userEvent.setup();
    renderDetail();

    await waitFor(() => expect(screen.getByRole('button', { name: 'Установить' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Установить' }));

    const depsDialog = await screen.findByRole('dialog');
    await user.click(within(depsDialog).getByRole('button', { name: 'Установить' }));

    await waitFor(() => expect(api.createModInstallRequest).toHaveBeenCalled());
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled(), { timeout: 5000 });
    await waitFor(() => expect(screen.getByText('Синхронизация с сервером')).toBeInTheDocument(), {
      timeout: 5000,
    });
  });

  it('shows installed badge when mod is already installed', async () => {
    vi.mocked(api.listInstanceResources).mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          project_id: 'sodium',
          project_name: 'Sodium',
          version_number: '0.5.0',
          filename: 'sodium.jar',
          resource_type: 'mod',
          installed_at: '2026-01-01T00:00:00Z',
        },
      ],
    });
    renderDetail();
    await waitFor(() => expect(screen.getByText('Установлен')).toBeInTheDocument());
  });

  it('shows not found when project missing', async () => {
    vi.mocked(api.getModProject).mockRejectedValueOnce(new Error('not found'));
    renderDetail();
    await waitFor(() => expect(testMessage.error).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByText('Мод не найден')).toBeInTheDocument());
  });

  it('shows empty versions state', async () => {
    vi.mocked(api.listModVersions).mockResolvedValue({ items: [] });
    renderDetail();
    await waitFor(() => expect(screen.getByText('Нет подходящих версий')).toBeInTheDocument());
  });
});
