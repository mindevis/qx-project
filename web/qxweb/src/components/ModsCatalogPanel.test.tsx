import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { ModsCatalogPanel } from '@/components/ModsCatalogPanel';
import { renderWithTheme } from '@/test/test-utils';

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
  client_side: 'required',
  server_side: 'required',
  project_type: 'mod' as const,
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
    renderCatalog();
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(api.browseMods).toHaveBeenCalled();
    expect(screen.getByRole('link', { name: 'Sodium' })).toHaveAttribute(
      'href',
      '/launcher/instances/inst-1/resources/catalog/modrinth/sodium',
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
    await user.click(screen.getByText('Модпаки'));
    await waitFor(() =>
      expect(api.browseMods).toHaveBeenLastCalledWith(
        expect.objectContaining({ type: 'modpack' }),
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
});
