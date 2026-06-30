import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
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
      <Route path="/launcher/instances/:instanceId/resources" element={<LauncherInstanceResourcesPage />} />
      <Route path="/launcher" element={<div>Launcher home</div>} />
    </Routes>,
    route,
  );
}

describe('LauncherInstanceResourcesPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.spyOn(api, 'listInstances').mockResolvedValue({ items: [forgeInstance] });
    vi.spyOn(api, 'searchMods').mockResolvedValue({ items: [], curseforge_enabled: true });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('renders resources panel for modded instance', async () => {
    renderResources();
    await waitFor(() => expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Forge'));
    expect(screen.getByLabelText('Ресурсы')).toBeInTheDocument();
    expect(screen.getByText('Назад к лаунчеру')).toBeInTheDocument();
  });

  it('redirects when instance is not found', async () => {
    vi.mocked(api.listInstances).mockResolvedValueOnce({ items: [] });
    renderResources();
    await waitFor(() => expect(screen.getByText('Launcher home')).toBeInTheDocument());
  });

  it('redirects vanilla instances to launcher home', async () => {
    vi.mocked(api.listInstances).mockResolvedValueOnce({
      items: [{ ...forgeInstance, loader: 'vanilla' }],
    });
    renderResources();
    await waitFor(() => expect(screen.getByText('Launcher home')).toBeInTheDocument());
  });

  it('searches mods from the resources page', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(api.searchMods).mockResolvedValueOnce({
      items: [
        {
          source: 'modrinth',
          id: 'sodium',
          name: 'Sodium',
          summary: 'Performance mod',
          external_url: 'https://modrinth.com/mod/sodium',
          client_side: 'unsupported',
          server_side: 'required',
        },
      ],
      curseforge_enabled: true,
    });

    renderResources();
    await waitFor(() => expect(screen.getByLabelText('Ресурсы')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Поиск по названию…'), 'sodium');
    await user.click(screen.getByRole('button', { name: 'Найти' }));
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
  });
});
