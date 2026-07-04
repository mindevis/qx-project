import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api } from '@/api/client';
import { INSTALLED_RESOURCES_VIEW_STORAGE_KEY } from '@/lib/installedResourcesView';
import { renderWithProviders } from '@/test/test-utils';
import { InstanceInstalledResources } from './InstanceInstalledResources';
import { InstanceModsProvider } from './InstanceModsContext';

const forgeInstance = {
  id: 'inst-1',
  name: 'Forge',
  mc_version: '1.21',
  loader: 'forge',
  loader_version: '47.0.0',
  created_at: 'now',
  updated_at: 'now',
};

const installed = {
  source: 'modrinth' as const,
  project_id: 'sodium',
  project_name: 'Sodium',
  version_number: '0.5.0',
  filename: 'sodium.jar',
  resource_type: 'mod' as const,
  installed_at: '2026-01-01T00:00:00Z',
};

function renderInstalledResources() {
  return renderWithProviders(
    <InstanceModsProvider instance={forgeInstance} canSync={false}>
      <InstanceInstalledResources />
    </InstanceModsProvider>,
  );
}

describe('InstanceInstalledResources', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.spyOn(api, 'listInstanceResources').mockResolvedValue({ items: [installed] });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('defaults to list view', async () => {
    renderInstalledResources();

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(document.querySelector('.qxmods-installed-list')).toBeInTheDocument();
    expect(document.querySelector('.launcher-resources-grid')).not.toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Список', checked: true })).toBeInTheDocument();
  });

  it('switches to card view and persists preference', async () => {
    const user = userEvent.setup({ delay: null });
    renderInstalledResources();

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    await user.click(screen.getByRole('radio', { name: 'Карточки' }));

    expect(document.querySelector('.launcher-resources-grid')).toBeInTheDocument();
    expect(document.querySelector('.qxmods-installed-list')).not.toBeInTheDocument();
    expect(window.localStorage.getItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY)).toBe('cards');
  });

  it('restores card view from localStorage', async () => {
    window.localStorage.setItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY, 'cards');
    renderInstalledResources();

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(document.querySelector('.launcher-resources-grid')).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Карточки', checked: true })).toBeInTheDocument();
  });

  it('filters installed resources with search and links to installed project details', async () => {
    const user = userEvent.setup({ delay: null });
    renderInstalledResources();

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());

    const input = screen.getByRole('textbox');
    await user.type(input, 'missing');
    await user.click(screen.getByRole('button', { name: /Найти|Search/ }));

    expect(screen.queryByText('Sodium')).not.toBeInTheDocument();
    expect(screen.getByText(/No results|Нет результатов|Ничего не найдено/i)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Сбросить поиск|Clear search/ }));
    await user.type(input, 'Sodium');
    await user.click(screen.getByRole('button', { name: /Найти|Search/ }));

    expect(screen.getByRole('link', { name: 'Sodium' })).toHaveAttribute(
      'href',
      '/launcher/instances/inst-1/resources/catalog/modrinth/sodium',
    );
  });
});
