import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Input, Space, Typography } from 'antd';
import { openServerConsole, type ConsoleMessage } from '@/api/client';

type ServerConsolePanelProps = {
  serverId: string;
  agentOnline: boolean;
};

export function ServerConsolePanel({ serverId, agentOnline }: ServerConsolePanelProps) {
  const [lines, setLines] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const [command, setCommand] = useState('');
  const preRef = useRef<HTMLPreElement>(null);
  const sessionRef = useRef<ReturnType<typeof openServerConsole> | null>(null);

  const appendLine = useCallback((line: string) => {
    setLines((prev) => [...prev.slice(-499), line]);
  }, []);

  useEffect(() => {
    let session: ReturnType<typeof openServerConsole> | null = null;
    const timer = window.setTimeout(() => {
      session = openServerConsole(serverId, {
        onMessage: (msg: ConsoleMessage) => {
          if (msg.type === 'output' && msg.line) {
            appendLine(`[${msg.stream ?? 'out'}] ${msg.line}`);
          }
          if (msg.type === 'status') {
            setConnected(msg.status === 'connected');
            if (msg.detail) {
              appendLine(`[status] ${msg.detail}`);
            } else if (msg.status === 'error') {
              appendLine('[status] ошибка консоли');
            }
          }
        },
        onClose: () => setConnected(false),
      });
      sessionRef.current = session;
    }, 0);

    return () => {
      window.clearTimeout(timer);
      session?.close();
      sessionRef.current = null;
      setConnected(false);
    };
  }, [serverId, appendLine]);

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
      <Typography.Text type={connected && agentOnline ? 'success' : 'secondary'}>
        {connected && agentOnline ? 'Консоль подключена' : 'Подключение…'}
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
        {lines.join('\n')}
      </pre>
      <Space.Compact style={{ width: '100%' }}>
        <Input
          placeholder="Команда сервера (Enter)"
          value={command}
          disabled={!connected || !agentOnline}
          onChange={(e) => setCommand(e.target.value)}
          onPressEnter={send}
        />
        <Button type="primary" disabled={!connected || !agentOnline} onClick={send}>
          Отправить
        </Button>
      </Space.Compact>
    </Space>
  );
}

/** Stop/Restart only when Minecraft process is actually running. */
export function shouldShowMinecraftControls(server: { minecraft_running?: boolean }): boolean {
  return server.minecraft_running === true;
}

/** Console is meaningful only after Start (or while a start attempt is in progress). */
export function shouldShowServerConsole(server: {
  status: string;
  minecraft_running?: boolean;
}): boolean {
  return (
    server.minecraft_running === true ||
    server.status === 'starting' ||
    server.status === 'error'
  );
}
