import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import * as apiClient from '@/api/client';
import { renderWithTheme, I18nThemeWrapper } from '@/test/test-utils';
import { ServerConsolePanel, shouldShowGameServerConsole, shouldShowMinecraftControls, shouldShowServerConsole } from './ServerConsolePanel';

class MockWebSocket {
  static OPEN = 1;
  static instances: MockWebSocket[] = [];
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  readyState = MockWebSocket.OPEN;
  sent: string[] = [];

  constructor(_url: string) {
    MockWebSocket.instances.push(this);
    queueMicrotask(() => {
      this.onmessage?.({
        data: JSON.stringify({ type: 'status', status: 'connected' }),
      });
    });
  }

  addEventListener(type: string, fn: () => void, _opts?: { once?: boolean }) {
    if (type === 'open') {
      queueMicrotask(fn);
    }
  }

  send(data: string) {
    this.sent.push(data);
  }

  close = vi.fn(function (this: MockWebSocket) {
    this.readyState = 3;
    this.onclose?.();
  });
}

describe('shouldShowMinecraftControls', () => {
  it('is hidden until minecraft process is running', () => {
    expect(shouldShowMinecraftControls({})).toBe(false);
    expect(shouldShowMinecraftControls({ minecraft_running: false })).toBe(false);
    expect(shouldShowMinecraftControls({ minecraft_running: true })).toBe(true);
  });
});

describe('shouldShowGameServerConsole', () => {
  it('shows console for provisioning and running game servers with online agent', () => {
    expect(shouldShowGameServerConsole({ status: 'stopped' }, true)).toBe(false);
    expect(shouldShowGameServerConsole({ status: 'running' }, false)).toBe(false);
    expect(shouldShowGameServerConsole({ status: 'installing' }, true)).toBe(true);
    expect(shouldShowGameServerConsole({ status: 'starting' }, true)).toBe(true);
    expect(shouldShowGameServerConsole({ status: 'running' }, true)).toBe(true);
    expect(shouldShowGameServerConsole({ status: 'error' }, true)).toBe(true);
  });
});

describe('shouldShowServerConsole', () => {
  it('is hidden until minecraft process is running', () => {
    expect(shouldShowServerConsole({})).toBe(false);
    expect(shouldShowServerConsole({ minecraft_running: false })).toBe(false);
    expect(shouldShowServerConsole({ status: 'starting' } as never)).toBe(false);
    expect(shouldShowServerConsole({ status: 'error' } as never)).toBe(false);
    expect(shouldShowServerConsole({ minecraft_running: true })).toBe(true);
  });
});

describe('ServerConsolePanel', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal('WebSocket', MockWebSocket);
    apiClient.saveTokens({
      access_token: 'token',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('shows status error detail', async () => {
    renderWithTheme(<ServerConsolePanel serverId="srv-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'status', status: 'error', detail: 'agent offline' }) });

    await waitFor(() => expect(screen.getByText(/\[status\] agent offline/)).toBeInTheDocument());
  });

  it('shows generic status error without detail', async () => {
    renderWithTheme(<ServerConsolePanel serverId="srv-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'status', status: 'error' }) });

    await waitFor(() =>
      expect(screen.getByText('[status] ошибка консоли')).toBeInTheDocument(),
    );
  });

  it('connects and sends console input', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<ServerConsolePanel serverId="srv-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    await user.type(screen.getByPlaceholderText('Команда сервера (Enter)'), 'list');
    await user.click(screen.getByRole('button', { name: 'Отправить' }));

    await waitFor(() => {
      expect(MockWebSocket.instances[0]?.sent.length).toBeGreaterThan(0);
    });
    expect(MockWebSocket.instances[0]?.sent[0]).toContain('list');
    expect(screen.getByText('> list')).toBeInTheDocument();
  });

  it('sends console input with game server id', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<ServerConsolePanel serverId="srv-1" gameServerId="gs-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    await user.type(screen.getByPlaceholderText('Команда сервера (Enter)'), 'list');
    await user.click(screen.getByRole('button', { name: 'Отправить' }));

    await waitFor(() => {
      expect(MockWebSocket.instances[0]?.sent.length).toBeGreaterThan(0);
    });
    expect(MockWebSocket.instances[0]?.sent[0]).toContain('list');
    expect(MockWebSocket.instances[0]?.sent[0]).toContain('gs-1');
  });

  it('renders streamed console output', async () => {
    renderWithTheme(<ServerConsolePanel serverId="srv-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'output', stream: 'err', line: 'boom' }) });
    ws?.onmessage?.({ data: JSON.stringify({ type: 'status', status: 'connected', detail: 'ready' }) });

    await waitFor(() => expect(screen.getByText(/\[err\] boom/)).toBeInTheDocument());
    expect(screen.getByText(/\[status\] ready/)).toBeInTheDocument();
  });

  it('reconnects when server id changes', async () => {
    const { rerender } = render(<ServerConsolePanel serverId="srv-1" agentOnline />, {
      wrapper: I18nThemeWrapper,
    });

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    expect(ws).toBeDefined();

    rerender(<ServerConsolePanel serverId="srv-2" agentOnline />);
    expect(ws?.close).toHaveBeenCalled();

    await waitFor(() => expect(MockWebSocket.instances.length).toBeGreaterThan(1));
  });

  it('filters output by game server id when provided', async () => {
    renderWithTheme(
      <ServerConsolePanel serverId="srv-1" gameServerId="gs-1" agentOnline />,
    );

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({
      data: JSON.stringify({ type: 'output', stream: 'out', line: 'other', game_server_id: 'gs-2' }),
    });
    ws?.onmessage?.({
      data: JSON.stringify({ type: 'output', stream: 'out', line: 'untagged' }),
    });
    ws?.onmessage?.({
      data: JSON.stringify({ type: 'output', stream: 'out', line: 'mine', game_server_id: 'gs-1' }),
    });

    await waitFor(() => expect(screen.getByText(/\[out\] mine/)).toBeInTheDocument());
    expect(screen.queryByText(/\[out\] other/)).not.toBeInTheDocument();
    expect(screen.queryByText(/\[out\] untagged/)).not.toBeInTheDocument();
  });

  it('renders output without stream as out', async () => {
    renderWithTheme(<ServerConsolePanel serverId="srv-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'output', line: 'plain' }) });

    await waitFor(() => expect(screen.getByText(/\[out\] plain/)).toBeInTheDocument());
  });

  it('scrolls console output when new lines arrive', async () => {
    const { container } = renderWithTheme(<ServerConsolePanel serverId="srv-1" agentOnline />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const pre = container.querySelector('pre');
    expect(pre).not.toBeNull();
    const scrollTo = vi.fn();
    Object.defineProperty(pre!, 'scrollTo', { value: scrollTo, configurable: true, writable: true });
    Object.defineProperty(pre!, 'scrollHeight', { value: 100, configurable: true });

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'output', stream: 'out', line: 'hello' }) });

    await waitFor(() => expect(screen.getByText(/\[out\] hello/)).toBeInTheDocument());
    await waitFor(() => expect(scrollTo).toHaveBeenCalledWith(0, 100));
  });

  it('ignores empty commands and closes on unmount', async () => {
    const user = userEvent.setup({ delay: null });
    const { unmount } = render(<ServerConsolePanel serverId="srv-1" agentOnline />, {
      wrapper: I18nThemeWrapper,
    });

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    await user.click(screen.getByRole('button', { name: 'Отправить' }));
    expect(MockWebSocket.instances[0]?.sent).toHaveLength(0);

    await user.type(screen.getByPlaceholderText('Команда сервера (Enter)'), 'help{enter}');
    await waitFor(() => expect(MockWebSocket.instances[0]?.sent[0]).toContain('help'));

    unmount();
    expect(MockWebSocket.instances[0]?.close).toHaveBeenCalled();
  });
});
