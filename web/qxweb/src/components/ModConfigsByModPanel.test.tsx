import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from 'antd';
import { testMessage } from '@/test/test-message';
import { renderWithTheme } from '@/test/test-utils';
import { ModConfigsByModPanel } from './ModConfigsByModPanel';

const fabricMod = {
  key: 'fabric-api',
  label: 'Fabric API',
  filename: 'fabric-api-0.100.0+1.21.jar',
  project_name: 'Fabric API',
};

const fileApi = {
  listDir: vi.fn(),
  readFile: vi.fn(),
  writeFile: vi.fn(),
};

describe('ModConfigsByModPanel', () => {
  beforeEach(() => {
    vi.spyOn(Modal, 'confirm').mockImplementation((config) => {
      config.onOk?.();
      return { destroy: vi.fn(), update: vi.fn() };
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows device note when instance bridge is unavailable', () => {
    renderWithTheme(
      <ModConfigsByModPanel mode="instance" available={false} mods={[]} fileApi={fileApi} />,
    );
    expect(screen.getByText(/QXLauncher/i)).toBeInTheDocument();
  });

  it('lists grouped config files and filters by search', async () => {
    const user = userEvent.setup({ delay: null });
    fileApi.listDir.mockImplementation(async (path: string) => {
      if (path === 'config') {
        return [
          { path: 'config/fabric-api.toml', dir: false, name: 'fabric-api.toml', size: 100 },
          { path: 'config/sodium-options.json', dir: false, name: 'sodium-options.json', size: 200 },
        ];
      }
      return [];
    });

    renderWithTheme(
      <ModConfigsByModPanel
        mode="instance"
        available
        mods={[fabricMod]}
        fileApi={fileApi}
        configSync={{
          instanceId: 'inst-1',
          instanceLoader: 'fabric',
          deviceLinked: true,
        }}
      />,
    );

    await waitFor(() => expect(screen.getAllByText('fabric-api.toml').length).toBeGreaterThan(0));
    expect(screen.getByText('100 B')).toBeInTheDocument();
    expect(screen.getByText(/Файлы: 2/i)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Прочее/i }));
    await waitFor(() => expect(screen.getAllByText('sodium-options.json').length).toBeGreaterThan(0));
    await user.type(screen.getByPlaceholderText(/Поиск мода или файла/i), 'sodium');
    await waitFor(() => expect(screen.queryByText('fabric-api.toml')).not.toBeInTheDocument());
    expect(screen.getAllByText('sodium-options.json').length).toBeGreaterThan(0);
  });

  it('tracks unsaved changes and saves edits', async () => {
    const user = userEvent.setup({ delay: null });
    fileApi.listDir.mockResolvedValue([
      { path: 'config/fabric-api.toml', dir: false, name: 'fabric-api.toml', size: 100 },
    ]);
    fileApi.readFile.mockResolvedValue('enabled=true');
    fileApi.writeFile.mockResolvedValue(undefined);

    renderWithTheme(
      <ModConfigsByModPanel mode="instance" available mods={[fabricMod]} fileApi={fileApi} />,
    );

    await waitFor(() => expect(screen.getAllByText('fabric-api.toml').length).toBeGreaterThan(0));
    await user.click(screen.getByRole('button', { name: /fabric-api\.toml/i }));
    await waitFor(() => expect(screen.getByDisplayValue('enabled=true')).toBeInTheDocument());

    const saveButton = screen.getByRole('button', { name: 'Сохранить' });
    expect(saveButton).toBeDisabled();

    await user.type(screen.getByRole('textbox'), '\nupdated=1');
    expect(screen.getByText(/несохран/i)).toBeInTheDocument();
    expect(saveButton).not.toBeDisabled();

    await user.click(saveButton);
    await waitFor(() => expect(testMessage.success).toHaveBeenCalled());
    expect(fileApi.writeFile).toHaveBeenCalledWith('config/fabric-api.toml', 'enabled=true\nupdated=1');
  });

  it('shows empty hint when no configs exist', async () => {
    fileApi.listDir.mockResolvedValue([]);

    renderWithTheme(
      <ModConfigsByModPanel mode="server" available mods={[]} fileApi={fileApi} />,
    );

    await waitFor(() => expect(screen.getByText('Конфигурационные файлы не найдены')).toBeInTheDocument());
    expect(screen.getByText(/Запустите сервер/i)).toBeInTheDocument();
  });

  it('does not ask to link QXLauncher on the game server', async () => {
    fileApi.listDir.mockResolvedValue([]);

    renderWithTheme(
      <ModConfigsByModPanel mode="server" available mods={[]} fileApi={fileApi} />,
    );

    await waitFor(() => expect(screen.getByText('Конфигурационные файлы не найдены')).toBeInTheDocument());
    expect(screen.queryByText(/Привяжите QXLauncher/i)).not.toBeInTheDocument();
  });
});
