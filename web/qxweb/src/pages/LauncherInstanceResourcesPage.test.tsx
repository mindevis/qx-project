import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Routes, Route } from 'react-router-dom';
import { api } from '@/api/client';
import { renderWithProviders } from '@/test/test-utils';
import { LauncherInstanceResourcesPage } from './LauncherInstanceResourcesPage';

const forgeInstance = {
  id: 'inst-1',
  name: 'Forge',
  mc_version: '1.21',
  loader: 'forge',
  loader_version: '47.0.0',
  created_at: 'now',
  updated_at: 'now',
};

function renderResources(route = '/launcher/instances/inst-1/resources') {
  return renderWithProviders(
    <Routes>
      <Route path="/launcher/instances/:instanceId/resources/*" element={<LauncherInstanceResourcesPage />} />
      <Route path="/launcher" element={<div>Launcher home</div>} />
    </Routes>,
    route,
  );
}

describe('LauncherInstanceResourcesPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.spyOn(api, 'listInstances').mockResolvedValue({ items: [forgeInstance] });
    vi.spyOn(api, 'listInstanceResources').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'browseMods').mockResolvedValue({
      items: [
        {
          source: 'modrinth',
          id: 'sodium',
          name: 'Sodium',
          summary: 'Performance mod',
          external_url: 'https://modrinth.com/mod/sodium',
          client_side: 'unsupported',
          server_side: 'required',
          project_type: 'mod',
        },
      ],
      has_more: false,
      curseforge_enabled: true,
    });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('renders installed resources view for modded instance', async () => {
    renderResources();
    await waitFor(() => expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Forge'));
    expect(screen.getByRole('navigation', { name: 'Разделы ресурсов инстанса' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Установленные/ })).toHaveClass('launcher-resources-tab--active');
    expect(screen.getByLabelText('Ресурсы')).toBeInTheDocument();
    expect(screen.getByText('Назад к лаунчеру')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText('Пока ничего не установлено')).toBeInTheDocument());
  });

  it('switches to catalog tab', async () => {
    const user = userEvent.setup({ delay: null });
    renderResources();
    await waitFor(() => expect(screen.getByRole('link', { name: /Каталог/ })).toBeInTheDocument());
    await user.click(screen.getByRole('link', { name: /Каталог/ }));
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(api.browseMods).toHaveBeenCalled();
  });

  it('redirects when instance is not found', async () => {
    vi.mocked(api.listInstances).mockResolvedValueOnce({ items: [] });
    renderResources();
    await waitFor(() => expect(screen.getByText('Launcher home')).toBeInTheDocument());
  });

  it('allows vanilla instances for datapacks', async () => {
    vi.mocked(api.listInstances).mockResolvedValueOnce({
      items: [{ ...forgeInstance, loader: 'vanilla' }],
    });
    renderResources();
    await waitFor(() => expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Forge'));
    expect(screen.getByLabelText('Ресурсы')).toBeInTheDocument();
  });
});
