import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { ModsCatalogPanel } from '@/components/ModsCatalogPanel';
import { clearModVersionCache } from '@/components/ModCatalogInstallControls';
import { renderWithTheme } from '@/test/test-utils';
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

const catalogItem = {
  source: 'modrinth' as const,
  id: 'sodium',
  name: 'Sodium',
  summary: 'Performance mod',
  external_url: 'https://modrinth.com/mod/sodium',
  icon_url: 'https://cdn.modrinth.com/data/AANobbMI/icon.png',
  client_side: 'required',
  server_side: 'required',
  project_type: 'mod' as const,
};

const modVersion = {
  id: 'ver-1',
  version_number: '0.5.0',
  game_versions: ['1.21'],
  loaders: ['forge'],
  files: [{ filename: 'sodium.jar', url: 'https://example.com/sodium.jar', primary: true, size: 1024 }],
};

function renderCatalog() {
  return renderWithTheme(
    <MemoryRouter initialEntries={['/']}>
      <InstanceModsProvider instance={forgeInstance} canSync={false}>
        <ModsCatalogPanel />
      </InstanceModsProvider>
    </MemoryRouter>,
  );
}

describe('ModsCatalogPanel', () => {
  beforeEach(() => {
    clearModVersionCache();
    vi.spyOn(api, 'listInstanceResources').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listModVersions').mockResolvedValue({ items: [modVersion] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [catalogItem],
      has_more: true,
      curseforge_enabled: true,
    });
    vi.spyOn(api, 'searchMods').mockResolvedValue({
      items: [catalogItem],
      curseforge_enabled: true,
    });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('loads browse catalog into table', async () => {
    const { container } = renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(screen.getByText('Клиент + сервер')).toBeInTheDocument();
    expect(api.browseMods).toHaveBeenCalled();
    expect(screen.getByRole('link', { name: 'Sodium' })).toHaveAttribute(
      'href',
      '/launcher/instances/inst-1/resources/catalog/modrinth/sodium',
    );
    const icon = container.querySelector('img.qxmods-catalog-table-icon');
    expect(icon).toHaveAttribute('src', catalogItem.icon_url);
    await waitFor(() => expect(api.listModVersions).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Установить' })).toBeInTheDocument(),
    );
  });

  it('runs search when query submitted', async () => {
    const user = userEvent.setup({ delay: null });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Необязательно: сузить по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() => expect(api.searchMods).toHaveBeenCalled());
  });

  it('loads more browse results', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(api.browseMods)
      .mockResolvedValueOnce({
        items: [catalogItem],
        has_more: true,
        curseforge_enabled: true,
      })
      .mockResolvedValueOnce({
        items: [{ ...catalogItem, id: 'lithium', name: 'Lithium' }],
        has_more: false,
        curseforge_enabled: true,
      });

    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Загрузить ещё' }));
    await waitFor(() => expect(screen.getByText('Lithium')).toBeInTheDocument());
    expect(api.browseMods).toHaveBeenCalledTimes(2);
  });

  it('switches resource type tab', async () => {
    const user = userEvent.setup({ delay: null });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    await user.click(screen.getByText('Ресурспаки'));
    await waitFor(() =>
      expect(api.browseMods).toHaveBeenLastCalledWith(
        expect.objectContaining({ type: 'resourcepack', loader: undefined }),
      ),
    );
  });

  it('includes datapack tab for modded instances', async () => {
    const user = userEvent.setup({ delay: null });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    await user.click(screen.getByText('Датапаки'));
    await waitFor(() =>
      expect(api.browseMods).toHaveBeenLastCalledWith(
        expect.objectContaining({ type: 'datapack', loader: undefined }),
      ),
    );
  });

  it('shows installed badge for installed catalog item', async () => {
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
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Установлен')).toBeInTheDocument());
  });

  it('opens server sync modal after catalog install', async () => {
    vi.spyOn(api, 'listServers').mockResolvedValue({
      items: [{ id: 'vps-1', name: 'VPS', slug: 'vps', status: 'running', agent_online: true }],
    });
    vi.spyOn(vpsGameServers, 'listVpsGameServers').mockResolvedValue([]);
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    const user = userEvent.setup();
    vi.spyOn(api, 'getModVersion').mockResolvedValue({ ...modVersion, dependencies: [] });
    vi.spyOn(api, 'createModInstallRequest').mockResolvedValue({
      id: 'req-1',
      instance_id: 'inst-1',
      status: 'queued',
      source: 'modrinth',
      project_id: 'sodium',
      project_name: 'Sodium',
      version_id: 'ver-1',
      filename: 'sodium.jar',
      resource_type: 'mod',
      expires_at: '2099-01-01T00:00:00Z',
    });
    vi.spyOn(api, 'getModInstallRequest').mockResolvedValue({
      id: 'req-1',
      instance_id: 'inst-1',
      status: 'completed',
      source: 'modrinth',
      project_id: 'sodium',
      project_name: 'Sodium',
      version_id: 'ver-1',
      filename: 'sodium.jar',
      resource_type: 'mod',
      expires_at: '2099-01-01T00:00:00Z',
    });

    renderWithTheme(
      <MemoryRouter initialEntries={['/']}>
        <InstanceModsProvider instance={forgeInstance} canSync>
          <ModsCatalogPanel />
        </InstanceModsProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(screen.getByRole('button', { name: 'Установить' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Установить' }));
    const depsDialog = await screen.findByRole('dialog');
    await user.click(within(depsDialog).getByRole('button', { name: 'Установить' }));

    await waitFor(() => expect(api.createModInstallRequest).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByText('Синхронизация с сервером')).toBeInTheDocument(), {
      timeout: 5000,
    });
  });
});
