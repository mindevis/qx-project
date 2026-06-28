import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { GameServerPropertiesPanel } from './GameServerPropertiesPanel';

describe('GameServerPropertiesPanel', () => {
  beforeEach(() => {
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows agent required when offline', () => {
    renderWithTheme(
      <GameServerPropertiesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={false} />,
    );
    expect(screen.getByText(/Deploy/i)).toBeInTheDocument();
  });

  it('edits boolean, numeric, and text properties', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'getVpsGameServerProperties').mockResolvedValue({
      properties: [
        { key: 'online-mode', value: 'false', boolean: true },
        { key: 'max-players', value: '20', boolean: false },
        { key: 'motd', value: 'Hello', boolean: false },
      ],
    });
    vi.spyOn(api, 'patchVpsGameServerProperties').mockResolvedValue({ status: 'ok' });

    renderWithTheme(
      <GameServerPropertiesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );

    await waitFor(() => expect(screen.getByText('motd')).toBeInTheDocument());

    await user.click(screen.getByRole('switch'));
    await waitFor(() => expect(message.success).toHaveBeenCalled());

    const textInput = screen.getByLabelText('motd');
    await user.clear(textInput);
    await user.type(textInput, 'Updated');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(message.success).toHaveBeenCalledTimes(2));
  });

  it('saves numeric and text properties from controls', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'getVpsGameServerProperties').mockResolvedValue({
      properties: [
        { key: 'max-players', value: '20', boolean: false },
        { key: 'motd', value: 'Hello', boolean: false },
      ],
    });
    vi.spyOn(api, 'patchVpsGameServerProperties').mockResolvedValue({ status: 'ok' });

    renderWithTheme(
      <GameServerPropertiesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByLabelText('max-players')).toBeInTheDocument());

    const playersInput = screen.getByRole('spinbutton', { name: 'max-players' });
    await user.click(playersInput);
    await user.keyboard('{ArrowUp}');
    await waitFor(() => expect(message.success).toHaveBeenCalled());

    const textInput = screen.getByLabelText('motd');
    await user.type(textInput, ' world');
    await user.keyboard('{Enter}');
    await waitFor(() => expect(message.success).toHaveBeenCalledTimes(2));
  });

  it('shows empty state and load errors', async () => {
    vi.spyOn(api, 'getVpsGameServerProperties')
      .mockResolvedValueOnce({ properties: [] })
      .mockRejectedValueOnce(new Error('props failed'));

    renderWithTheme(
      <GameServerPropertiesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() =>
      expect(screen.getByText(/server.properties пуст или недоступен/i)).toBeInTheDocument(),
    );

    renderWithTheme(
      <GameServerPropertiesPanel vpsId="srv-1" gameServerId="gs-2" agentOnline={true} />,
    );
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('props failed'));
  });

  it('reloads after save error', async () => {
    const user = userEvent.setup({ delay: null });
    const getSpy = vi
      .spyOn(api, 'getVpsGameServerProperties')
      .mockResolvedValueOnce({
        properties: [{ key: 'motd', value: 'Hello', boolean: false }],
      })
      .mockResolvedValueOnce({
        properties: [{ key: 'motd', value: 'Hello', boolean: false }],
      });
    vi.spyOn(api, 'patchVpsGameServerProperties').mockRejectedValue(new Error('save failed'));

    renderWithTheme(
      <GameServerPropertiesPanel vpsId="srv-1" gameServerId="gs-1" agentOnline={true} />,
    );
    await waitFor(() => expect(screen.getByLabelText('motd')).toBeInTheDocument());
    const input = screen.getByLabelText('motd');
    await user.clear(input);
    await user.type(input, 'Fail');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('save failed'));
    expect(getSpy).toHaveBeenCalledTimes(2);
  });
});
