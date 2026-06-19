import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import * as apiClient from '@/api/client';
import { ServerConsolePanel } from './ServerConsolePanel';

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

  send(data: string) {
    this.sent.push(data);
  }

  close = vi.fn(function (this: MockWebSocket) {
    this.readyState = 3;
    this.onclose?.();
  });
}

describe('ServerConsolePanel', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal('WebSocket', MockWebSocket);
    apiClient.saveTokens({
      access_token: 'token',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('connects and sends console input', async () => {
    const user = userEvent.setup({ delay: null });
    render(<ServerConsolePanel serverId="srv-1" enabled />);

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

  it('does not connect when disabled', () => {
    render(<ServerConsolePanel serverId="srv-1" enabled={false} />);
    expect(MockWebSocket.instances).toHaveLength(0);
    expect(screen.getByText('Консоль отключена — agent должен быть online')).toBeInTheDocument();
  });

  it('renders streamed console output', async () => {
    render(<ServerConsolePanel serverId="srv-1" enabled />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'output', stream: 'err', line: 'boom' }) });
    ws?.onmessage?.({ data: JSON.stringify({ type: 'status', status: 'connected', detail: 'ready' }) });

    await waitFor(() => expect(screen.getByText(/\[err\] boom/)).toBeInTheDocument());
    expect(screen.getByText(/\[status\] ready/)).toBeInTheDocument();
  });

  it('closes websocket session when console is disabled', async () => {
    const { rerender } = render(<ServerConsolePanel serverId="srv-1" enabled />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    expect(ws).toBeDefined();

    rerender(<ServerConsolePanel serverId="srv-1" enabled={false} />);
    expect(ws?.close).toHaveBeenCalled();
    expect(screen.getByText('Консоль отключена — agent должен быть online')).toBeInTheDocument();
  });

  it('renders output without stream as out', async () => {
    render(<ServerConsolePanel serverId="srv-1" enabled />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'output', line: 'plain' }) });

    await waitFor(() => expect(screen.getByText(/\[out\] plain/)).toBeInTheDocument());
  });

  it('scrolls console output when new lines arrive', async () => {
    render(<ServerConsolePanel serverId="srv-1" enabled />);

    await waitFor(() =>
      expect(screen.getByText('Консоль подключена')).toBeInTheDocument(),
    );

    const pre = document.querySelector('pre');
    expect(pre).not.toBeNull();
    const scrollTo = vi.fn();
    Object.defineProperty(pre!, 'scrollTo', { value: scrollTo, configurable: true });

    const ws = MockWebSocket.instances.at(-1);
    ws?.onmessage?.({ data: JSON.stringify({ type: 'output', stream: 'out', line: 'hello' }) });

    await waitFor(() => expect(screen.getByText(/\[out\] hello/)).toBeInTheDocument());
    expect(scrollTo).toHaveBeenCalledWith(0, pre!.scrollHeight);
  });

  it('ignores empty commands and closes on unmount', async () => {
    const user = userEvent.setup({ delay: null });
    const { unmount } = render(<ServerConsolePanel serverId="srv-1" enabled />);

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
