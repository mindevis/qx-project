import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerFilesPanel } from './GameServerFilesPanel';

describe('GameServerFilesPanel', () => {
  beforeEach(() => {
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows agent required when offline', () => {
    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={false} />,
    );
    expect(screen.getByText(/Deploy/i)).toBeInTheDocument();
  });

  it('lists directories and opens files', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockImplementation(async (vpsId, gameServerId, path) => {
      if (path === 'config') {
        return { items: [{ name: 'inside.txt', path: 'config/inside.txt', dir: false, size: 1 }] };
      }
      return {
        items: [
          { name: 'config', path: 'config', dir: true },
          { name: 'server.properties', path: 'server.properties', dir: false, size: 512 },
          { name: 'tiny.txt', path: 'tiny.txt', dir: false, size: 10 },
          { name: 'big.txt', path: 'big.txt', dir: false, size: 2 * 1024 * 1024 },
        ],
      };
    });
    vi.spyOn(api, 'readVpsGameServerFile').mockResolvedValue({ content: 'motd=Hello', path: 'server.properties' });
    vi.spyOn(api, 'writeVpsGameServerFile').mockResolvedValue({ status: 'ok' });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );

    await waitFor(() => expect(screen.getByText('server.properties')).toBeInTheDocument());
    expect(screen.getByText('512 B')).toBeInTheDocument();

    await user.click(screen.getByText('server.properties'));
    await waitFor(() => expect(screen.getByDisplayValue('motd=Hello')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /К списку файлов/i }));
    await waitFor(() => expect(screen.getByText('config')).toBeInTheDocument());
    await user.click(screen.getByText('config'));
    await waitFor(() => expect(screen.getByText('inside.txt')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: '/' }));
    await waitFor(() => expect(screen.getByText('server.properties')).toBeInTheDocument());
    expect(screen.getByText('10 B')).toBeInTheDocument();
    expect(screen.getByText('2.0 MB')).toBeInTheDocument();

    vi.spyOn(api, 'readVpsGameServerFile').mockResolvedValue({ content: 'tiny', path: 'tiny.txt' });
    await user.click(screen.getByText('tiny.txt'));
    await waitFor(() => expect(screen.getByDisplayValue('tiny')).toBeInTheDocument());
    await user.type(screen.getByRole('textbox'), '-edit');
    expect(screen.getByDisplayValue('tiny-edit')).toBeInTheDocument();
  });

  it('shows empty directory listing', async () => {
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({ items: [] });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByText('Папка пуста')).toBeInTheDocument());
  });

  it('shows read errors', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({
      items: [{ name: 'bad.txt', path: 'bad.txt', dir: false, size: 1 }],
    });
    vi.spyOn(api, 'readVpsGameServerFile').mockRejectedValue(new Error('open failed'));

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByText('bad.txt')).toBeInTheDocument());
    await user.click(screen.getByText('bad.txt'));
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('open failed'));
  });

  it('shows list load error', async () => {
    vi.spyOn(api, 'listVpsGameServerFiles').mockRejectedValue(new Error('list failed'));

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('list failed'));
  });

  it('deletes a file after confirmation', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles')
      .mockResolvedValueOnce({
        items: [{ name: 'eula.txt', path: 'eula.txt', dir: false, size: 10 }],
      })
      .mockResolvedValueOnce({ items: [] });
    const deleteSpy = vi.spyOn(api, 'deleteVpsGameServerFile').mockResolvedValue({ status: 'ok' });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByText('eula.txt')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Удалить' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Удалить' }));
    await waitFor(() =>
      expect(deleteSpy).toHaveBeenCalledWith('srv-1', 'gs-1', 'eula.txt'),
    );
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByText('Папка пуста')).toBeInTheDocument());
  });

  it('deletes a folder after confirmation without opening it', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles')
      .mockResolvedValueOnce({
        items: [{ name: 'world', path: 'world', dir: true }],
      })
      .mockResolvedValueOnce({ items: [] });
    const deleteSpy = vi.spyOn(api, 'deleteVpsGameServerFile').mockResolvedValue({ status: 'ok' });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByText('world')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Удалить' }));
    expect(screen.getByText('Удалить папку?')).toBeInTheDocument();
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Удалить' }));
    await waitFor(() => expect(deleteSpy).toHaveBeenCalledWith('srv-1', 'gs-1', 'world'));
    expect(api.listVpsGameServerFiles).toHaveBeenCalledWith('srv-1', 'gs-1', '');
  });

  it('shows save error', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({
      items: [{ name: 'a.txt', path: 'a.txt', dir: false, size: 1 }],
    });
    vi.spyOn(api, 'readVpsGameServerFile').mockResolvedValue({ content: 'x', path: 'a.txt' });
    vi.spyOn(api, 'writeVpsGameServerFile').mockRejectedValue(new Error('save failed'));

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByText('a.txt')).toBeInTheDocument());
    await user.click(screen.getByText('a.txt'));
    await waitFor(() => expect(screen.getByDisplayValue('x')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(testMessage.error).toHaveBeenCalledWith('save failed'));
  });
});
