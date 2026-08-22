import { describe, expect, it, vi, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerLaunchSettingsPanel } from './GameServerLaunchSettingsPanel';
import type { VpsGameServer } from '@/lib/vpsGameServers';

const game: VpsGameServer = {
  id: 'gs-1',
  name: 'Paper',
  server_type: 'paper',
  mc_version: '1.21',
  status: 'stopped',
  min_memory_mb: 1024,
  max_memory_mb: 2048,
  extra_jvm_args: ['-XX:+UseG1GC'],
  extra_args: [],
  created_at: 'now',
};

describe('GameServerLaunchSettingsPanel', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('saves RAM and extra launch arguments', async () => {
    const user = userEvent.setup({ delay: null });
    const onUpdated = vi.fn();
    const patch = vi.spyOn(api, 'updateVpsGameServer').mockResolvedValue({
      ...game,
      min_memory_mb: 2048,
      max_memory_mb: 4096,
      extra_jvm_args: ['-XX:+UseG1GC', '-XX:+ParallelRefProcEnabled'],
      extra_args: ['--forceUpgrade'],
    });

    renderWithTheme(
      <GameServerLaunchSettingsPanel vpsId="srv-1" game={game} onUpdated={onUpdated} />,
    );

    expect(screen.getByText('Параметры запуска')).toBeInTheDocument();

    const memoryInputs = screen.getAllByRole('spinbutton');
    await user.clear(memoryInputs[0]!);
    await user.type(memoryInputs[0]!, '2048');
    await user.clear(memoryInputs[1]!);
    await user.type(memoryInputs[1]!, '4096');

    const areas = screen.getAllByRole('textbox');
    await user.clear(areas[0]!);
    await user.type(areas[0]!, '-XX:+UseG1GC{enter}-XX:+ParallelRefProcEnabled');
    await user.clear(areas[1]!);
    await user.type(areas[1]!, '--forceUpgrade');

    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(patch).toHaveBeenCalled());
    expect(patch).toHaveBeenCalledWith('srv-1', 'gs-1', {
      min_memory_mb: 2048,
      max_memory_mb: 4096,
      extra_jvm_args: ['-XX:+UseG1GC', '-XX:+ParallelRefProcEnabled'],
      extra_args: ['--forceUpgrade'],
    });
    expect(onUpdated).toHaveBeenCalled();
    expect(testMessage.success).toHaveBeenCalled();
  });

  it('prefills Aikar flags when extra JVM args are empty', () => {
    renderWithTheme(
      <GameServerLaunchSettingsPanel
        vpsId="srv-1"
        game={{ ...game, extra_jvm_args: [] }}
        onUpdated={vi.fn()}
      />,
    );
    expect(screen.getByDisplayValue(/AlwaysPreTouch/)).toBeInTheDocument();
  });
});
