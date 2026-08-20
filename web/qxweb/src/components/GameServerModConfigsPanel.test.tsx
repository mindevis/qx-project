import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerModConfigsPanel } from './GameServerModConfigsPanel';

describe('GameServerModConfigsPanel', () => {
  beforeEach(() => {
    vi.spyOn(api, 'listVpsGameServerMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerClientMods').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listMonitoringBindings').mockResolvedValue({ items: [] });
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({ items: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('switches to client configs and shows the upload actions', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockImplementation(async (_vpsId, _gameServerId, path) => {
      if (path === 'client-config') {
        return {
          items: [
            {
              name: 'sodium-options.json',
              path: 'client-config/sodium-options.json',
              dir: false,
              size: 40,
            },
          ],
        };
      }
      return { items: [] };
    });

    renderWithTheme(
      <GameServerModConfigsPanel vpsId="srv-1" gameServerId="gs-1" agentOnline loader="fabric" />,
    );

    await waitFor(() => expect(screen.getByText('Конфигурационные файлы не найдены')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: /Загрузить файлы/i })).not.toBeInTheDocument();

    await user.click(screen.getByText('Клиентские'));
    await waitFor(() => expect(screen.getAllByText('sodium-options.json').length).toBeGreaterThan(0));
    expect(screen.getByRole('button', { name: /Загрузить файлы/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Загрузить папку/i })).toBeInTheDocument();
  });
});
