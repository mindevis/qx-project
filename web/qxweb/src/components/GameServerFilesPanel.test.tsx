import { describe, expect, it, vi, afterEach } from 'vitest';
import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from 'antd';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerFilesPanel, isValidFileManagerName, joinGameServerRelPath } from './GameServerFilesPanel';

describe('GameServerFilesPanel', () => {
  afterEach(() => {
    Modal.destroyAll();
    cleanup();
    vi.restoreAllMocks();
  });

  it('joins relative paths and validates names', () => {
    expect(joinGameServerRelPath('', 'eula.txt')).toBe('eula.txt');
    expect(joinGameServerRelPath('plugins', 'LuckPerms.jar')).toBe('plugins/LuckPerms.jar');
    expect(isValidFileManagerName('config')).toBe(true);
    expect(isValidFileManagerName('a/b')).toBe(false);
    expect(isValidFileManagerName('..')).toBe(false);
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
    expect(screen.getByRole('button', { name: 'Новый файл' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Новая папка' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Загрузить' })).toBeInTheDocument();
  });

  it('creates a folder in the current directory', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({ items: [] });
    const mkdir = vi.spyOn(api, 'mkdirVpsGameServerFile').mockResolvedValue({
      status: 'ok',
      path: 'plugins',
    });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByRole('button', { name: 'Новая папка' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Новая папка' }));
    await user.type(screen.getByPlaceholderText('config'), 'plugins');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(mkdir).toHaveBeenCalledWith('srv-1', 'gs-1', 'plugins'));
  });

  it('creates a file and opens the editor', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({ items: [] });
    const write = vi.spyOn(api, 'writeVpsGameServerFile').mockResolvedValue({ status: 'ok' });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByRole('button', { name: 'Новый файл' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Новый файл' }));
    await user.type(screen.getByPlaceholderText('notes.txt'), 'eula.txt');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(write).toHaveBeenCalledWith('srv-1', 'gs-1', 'eula.txt', ''));
    await waitFor(() => expect(screen.getByText('eula.txt')).toBeInTheDocument());
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('uploads a file into the opened folder', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockImplementation(async (_vpsId, _gameServerId, path) => {
      if (path === 'plugins') {
        return { items: [] };
      }
      return { items: [{ name: 'plugins', path: 'plugins', dir: true }] };
    });
    const upload = vi.spyOn(api, 'uploadVpsGameServerFile').mockResolvedValue({
      status: 'ok',
      path: 'plugins/LuckPerms.jar',
      filename: 'LuckPerms.jar',
    });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByText('plugins')).toBeInTheDocument());
    await user.click(screen.getByText('plugins'));
    await waitFor(() => expect(screen.getByText('Папка пуста')).toBeInTheDocument());

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
    const jar = new File(['jar-bytes'], 'LuckPerms.jar', { type: 'application/java-archive' });
    await user.upload(fileInput, jar);
    await waitFor(() =>
      expect(upload).toHaveBeenCalledWith('srv-1', 'gs-1', 'plugins', jar),
    );
  });

  it('rejects a name with slashes', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'listVpsGameServerFiles').mockResolvedValue({ items: [] });
    const mkdir = vi.spyOn(api, 'mkdirVpsGameServerFile').mockResolvedValue({
      status: 'ok',
      path: 'bad',
    });

    renderWithTheme(
      <GameServerFilesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await user.click(await screen.findByRole('button', { name: 'Новая папка' }));
    await user.type(screen.getByPlaceholderText('config'), 'a/b');
    await user.click(screen.getByRole('button', { name: 'Создать' }));
    await waitFor(() => expect(testMessage.error).toHaveBeenCalled());
    expect(mkdir).not.toHaveBeenCalled();
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
    const fileConfirm = await screen.findByRole('tooltip');
    await user.click(within(fileConfirm).getByRole('button', { name: 'Удалить' }));
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
    const folderConfirm = await screen.findByRole('tooltip');
    expect(within(folderConfirm).getByText('Удалить папку?')).toBeInTheDocument();
    await user.click(within(folderConfirm).getByRole('button', { name: 'Удалить' }));
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
