import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor } from '@testing-library/react';
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
  author: 'jellysquid3',
  downloads: 1_200_000,
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
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('loads browse catalog into table', async () => {
    const user = userEvent.setup({ delay: null });
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
    expect(screen.getByText('jellysquid3')).toBeInTheDocument();
    expect(screen.getByText('1.2M')).toBeInTheDocument();
    await user.click(screen.getByRole('combobox', { name: 'Версия' }));
    await waitFor(() => expect(api.listModVersions).toHaveBeenCalled());
    await waitFor(() => expect(screen.getAllByText('0.5.0').length).toBeGreaterThan(0));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Установить' })).toBeInTheDocument(),
    );
  });

  it('merges same-name mods from both sources into one row', async () => {
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          ...catalogItem,
          id: 'sodium',
          source: 'modrinth',
          name: 'Sodium',
          downloads: 1_200_000,
        },
        {
          ...catalogItem,
          id: '394468',
          source: 'curseforge',
          name: 'Sodium',
          downloads: 400_000,
          external_url: 'https://www.curseforge.com/minecraft/mc-mods/sodium',
        },
      ],
      has_more: false,
      curseforge_enabled: true,
    });

    renderCatalog();
    await waitFor(() => expect(screen.getByRole('link', { name: 'Sodium' })).toBeInTheDocument());
    expect(screen.getAllByRole('link', { name: 'Sodium' })).toHaveLength(1);
    expect(screen.getByRole('radio', { name: 'Modrinth' })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'CurseForge' })).toBeInTheDocument();
  });

  it('runs search when query submitted', async () => {
    const user = userEvent.setup({ delay: null });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Необязательно: сузить по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() =>
      expect(api.searchMods).toHaveBeenCalledWith(
        expect.objectContaining({ q: 'sodium', source: 'all' }),
      ),
    );
  });

  it('debounces catalog search input', async () => {
    const user = userEvent.setup({ delay: null });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Необязательно: сузить по названию…'), 'jei');
    expect(api.searchMods).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(api.searchMods).toHaveBeenCalledWith(expect.objectContaining({ q: 'jei' })),
    );
  });

  it('keeps previous rows visible while filters reload', async () => {
    const user = userEvent.setup({ delay: null });
    let finishReload: ((value: Awaited<ReturnType<typeof api.browseMods>>) => void) | undefined;
    vi.mocked(api.browseMods)
      .mockResolvedValueOnce({
        items: [catalogItem],
        has_more: false,
        curseforge_enabled: true,
      })
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishReload = resolve;
          }),
      );

    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    const [sourceSelect] = screen.getAllByRole('combobox');
    await user.click(sourceSelect!);
    await user.click(await screen.findByText('CurseForge'));
    expect(screen.getByText('Sodium')).toBeInTheDocument();
    finishReload?.({
      items: [{ ...catalogItem, id: 'jei', name: 'JEI' }],
      has_more: false,
      curseforge_enabled: true,
    });
    await waitFor(() => expect(screen.getByText('JEI')).toBeInTheDocument());
  });

  it('searches only selected source', async () => {
    const user = userEvent.setup({ delay: null });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());

    const [sourceSelect] = screen.getAllByRole('combobox');
    await user.click(sourceSelect!);
    await user.click(await screen.findByText('CurseForge'));
    await user.type(screen.getByPlaceholderText('Необязательно: сузить по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));

    await waitFor(() =>
      expect(api.searchMods).toHaveBeenCalledWith(
        expect.objectContaining({ q: 'sodium', source: 'curseforge' }),
      ),
    );
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

  it('hides installed catalog items by default', async () => {
    vi.mocked(api.browseMods).mockResolvedValue({
      items: [catalogItem, { ...catalogItem, id: 'lithium', name: 'Lithium' }],
      has_more: false,
      curseforge_enabled: true,
    });
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
    await waitFor(() => expect(screen.getByText('Lithium')).toBeInTheDocument());
    expect(screen.queryByText('Sodium')).not.toBeInTheDocument();
  });

  it('hides catalog items already copied from a game server as uploads', async () => {
    vi.mocked(api.browseMods).mockResolvedValue({
      items: [
        {
          ...catalogItem,
          id: '250498',
          source: 'curseforge',
          slug: 'mowzies-mobs',
          name: "Mowzie's Mobs",
        },
        { ...catalogItem, id: 'lithium', name: 'Lithium', slug: 'lithium' },
      ],
      has_more: false,
      curseforge_enabled: true,
    });
    vi.mocked(api.listInstanceResources).mockResolvedValue({
      items: [
        {
          source: 'upload',
          project_name: "Mowzie's Mobs-1.20.1-1.7.3",
          filename: "Mowzie's Mobs-1.20.1-1.7.3.jar",
          resource_type: 'mod',
          installed_at: '2026-01-01T00:00:00Z',
        },
      ],
    });
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Lithium')).toBeInTheDocument());
    expect(screen.queryByText("Mowzie's Mobs")).not.toBeInTheDocument();
  });

  it('shows only installed mods when toggle is enabled', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(api.browseMods).mockResolvedValue({
      items: [catalogItem, { ...catalogItem, id: 'lithium', name: 'Lithium' }],
      has_more: false,
      curseforge_enabled: true,
    });
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
    await waitFor(() => expect(screen.getByText('Lithium')).toBeInTheDocument());

    await user.click(screen.getByRole('switch', { name: 'Только установленные' }));

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(screen.queryByText('Lithium')).not.toBeInTheDocument();
    expect(screen.getByText('Установлен')).toBeInTheDocument();
  });

  it('opens server sync modal after catalog install', async () => {
    vi.spyOn(api, 'listServers').mockResolvedValue({
      items: [{ id: 'vps-1', name: 'VPS', slug: 'vps', status: 'running', agent_online: true }],
    });
    vi.spyOn(vpsGameServers, 'listVpsGameServers').mockResolvedValue([]);
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    const user = userEvent.setup();
    vi.spyOn(api, 'getModVersion').mockResolvedValue({ ...modVersion, dependencies: [] });
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

    await waitFor(() => expect(api.createModInstallRequest).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByText('Синхронизация с сервером')).toBeInTheDocument(), {
      timeout: 5000,
    });
  });
});
