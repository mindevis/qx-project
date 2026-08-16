import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from 'antd';
import { testMessage } from '@/test/test-message';
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
    vi.spyOn(api, 'listInstanceResources').mockResolvedValue({ items: [installed] });
  });

  afterEach(() => {
    Modal.destroyAll();
    cleanup();
    vi.restoreAllMocks();
  });

  it('shows resources panel for vanilla instances (datapacks)', async () => {
    vi.mocked(api.listInstanceResources).mockResolvedValueOnce({ items: [] });
    renderWithProviders(
      <InstanceResourcesPanel
        instance={{ ...forgeInstance, loader: 'vanilla' }}
        canSync={false}
      />,
    );
    await waitFor(() => expect(screen.getByLabelText('Ресурсы')).toBeInTheDocument());
    await waitFor(() =>
      expect(screen.getByText('Пока ничего не установлено')).toBeInTheDocument(),
    );
  });

  it('shows installed resources', async () => {
    renderWithProviders(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);

    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());
    expect(api.listInstanceResources).toHaveBeenCalledWith('inst-1');
    expect(screen.queryByRole('link', { name: /Добавить/ })).not.toBeInTheDocument();
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
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('load failed'));
  });

  it('removes installed resource after confirmation', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'deleteInstanceResource').mockResolvedValue(undefined);
    renderWithProviders(<InstanceResourcesPanel instance={forgeInstance} canSync={false} />);
    await waitFor(() => expect(screen.getByText('Sodium')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: 'Удалить' }));
    const confirm = await screen.findByRole('tooltip');
    await user.click(within(confirm).getByRole('button', { name: 'Удалить' }));

    await waitFor(() => expect(api.deleteInstanceResource).toHaveBeenCalledWith('inst-1', {
      source: 'modrinth',
      project_id: 'sodium',
      filename: 'sodium.jar',
      resource_type: 'mod',
    }));
    await waitFor(() => expect(screen.queryByText('Sodium')).not.toBeInTheDocument());
  });
});
