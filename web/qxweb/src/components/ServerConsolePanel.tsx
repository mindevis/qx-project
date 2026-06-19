import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Input, Space, Typography } from 'antd';
import { openServerConsole, type ConsoleMessage } from '@/api/client';

type ServerConsolePanelProps = {
  serverId: string;
  enabled: boolean;
};

export function ServerConsolePanel({ serverId, enabled }: ServerConsolePanelProps) {
  const [lines, setLines] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const [command, setCommand] = useState('');
  const preRef = useRef<HTMLPreElement>(null);
  const sessionRef = useRef<ReturnType<typeof openServerConsole> | null>(null);

  const appendLine = useCallback((line: string) => {
    setLines((prev) => [...prev.slice(-499), line]);
  }, []);

  useEffect(() => {
    if (!enabled) {
      /* v8 ignore next */
      sessionRef.current?.close();
      sessionRef.current = null;
      setConnected(false);
      return;
    }

    const session = openServerConsole(serverId, {
      onMessage: (msg: ConsoleMessage) => {
        if (msg.type === 'output' && msg.line) {
          appendLine(`[${msg.stream ?? 'out'}] ${msg.line}`);
        }
        if (msg.type === 'status') {
          setConnected(msg.status === 'connected');
          if (msg.detail) {
            appendLine(`[status] ${msg.detail}`);
          }
        }
      },
      onClose: () => setConnected(false),
    });
    sessionRef.current = session;

    return () => {
      session.close();
      sessionRef.current = null;
    };
  }, [enabled, serverId, appendLine]);

  useEffect(() => {
    const el = preRef.current;
    if (el && typeof el.scrollTo === 'function') {
      el.scrollTo(0, el.scrollHeight);
    }
  }, [lines]);

  const send = () => {
    const line = command.trim();
    if (!line || !sessionRef.current) return;
    sessionRef.current.send(line);
    appendLine(`> ${line}`);
    setCommand('');
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Typography.Text type={connected ? 'success' : 'secondary'}>
        {connected ? 'Консоль подключена' : 'Консоль отключена — agent должен быть online'}
      </Typography.Text>
      <pre
        ref={preRef}
        style={{
          margin: 0,
          padding: 12,
          minHeight: 240,
          maxHeight: 360,
          overflow: 'auto',
          background: '#1e1e1e',
          color: '#d4d4d4',
          borderRadius: 8,
          fontSize: 13,
          fontFamily: 'Consolas, monospace',
        }}
      >
        {lines.length === 0 ? 'Ожидание вывода…' : lines.join('\n')}
      </pre>
      <Space.Compact style={{ width: '100%' }}>
        <Input
          placeholder="Команда сервера (Enter)"
          value={command}
          disabled={!connected}
          onChange={(e) => setCommand(e.target.value)}
          onPressEnter={send}
        />
        <Button type="primary" disabled={!connected} onClick={send}>
          Отправить
        </Button>
      </Space.Compact>
    </Space>
  );
}
