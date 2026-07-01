import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { message } from 'antd';
import { api } from '@/api/client';
import { renderWithProviders } from '@/test/test-utils';
import { InstanceResourcesPanel } from './InstanceResourcesPanel';

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

describe('InstanceResourcesPanel', () => {
  beforeEach(() => {
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'listInstanceResources').mockResolvedValue({ items: [installed] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns null for vanilla instances', () => {
    renderWithProviders(
      <InstanceResourcesPanel
        instance={{ ...forgeInstance, loader: 'vanilla' }}
        canSync={false}
      />,
    );
    expect(screen.queryByLabelText('Ресурсы')).not.toBeInTheDocument();
  });

  it('shows installed resources and add button', async () => {
    renderWithProviders(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(api.listInstanceResources).toHaveBeenCalledWith('inst-1');
    expect(screen.getByRole('link', { name: /Добавить/ })).toBeInTheDocument();
  });

  it('shows empty state when nothing installed', async () => {
    vi.mocked(api.listInstanceResources).mockResolvedValueOnce({ items: [] });
    renderWithProviders(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);
    await waitFor(() =>
      expect(screen.getByText('Пока ничего не установлено')).toBeInTheDocument(),
    );
  });

  it('reports load failures', async () => {
    vi.mocked(api.listInstanceResources).mockRejectedValueOnce(new Error('load failed'));
    renderWithProviders(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('load failed'));
  });
});
