import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Input, Space, Typography } from 'antd';
import { openServerConsole, type ConsoleMessage } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';

type ServerConsolePanelProps = {
  serverId: string;
  /** When set, only lines tagged for this game server instance are shown. */
  gameServerId?: string;
  agentOnline: boolean;
  /** Smaller output area when embedded in a game server card. */
  embedded?: boolean;
};

export function ServerConsolePanel({
  serverId,
  gameServerId,
  agentOnline,
  embedded = false,
}: ServerConsolePanelProps) {
  const { t } = useI18n();
  const [lines, setLines] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const [command, setCommand] = useState('');
  const preRef = useRef<HTMLPreElement>(null);
  const sessionRef = useRef<ReturnType<typeof openServerConsole> | null>(null);

  const appendLine = useCallback((line: string) => {
    setLines((prev) => [...prev.slice(-499), line]);
  }, []);

  const handleMessage = useCallback(
    (msg: ConsoleMessage) => {
      if (gameServerId && msg.type === 'output' && msg.game_server_id !== gameServerId) {
        return;
      }
      if (msg.type === 'output' && msg.line != null && msg.line !== '') {
        appendLine(`[${msg.stream ?? 'out'}] ${msg.line}`);
      }
      if (msg.type === 'status') {
        setConnected(msg.status === 'connected');
        if (msg.detail) {
          appendLine(`[status] ${msg.detail}`);
        } else if (msg.status === 'error') {
          appendLine(`[status] ${t('console.error')}`);
        }
      }
    },
    [appendLine, gameServerId, t],
  );

  const handleMessageRef = useRef(handleMessage);
  handleMessageRef.current = handleMessage;

  useEffect(() => {
    setLines([]);
    let session: ReturnType<typeof openServerConsole> | null = null;
    const timer = window.setTimeout(() => {
      session = openServerConsole(
        serverId,
        {
          onMessage: (msg) => handleMessageRef.current(msg),
          onClose: () => setConnected(false),
        },
        gameServerId,
      );
      sessionRef.current = session;
    }, 0);

    return () => {
      window.clearTimeout(timer);
      session?.close();
      sessionRef.current = null;
      setConnected(false);
    };
  }, [serverId, gameServerId]);

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
    <div className={`servers-console${embedded ? ' servers-console--embedded' : ''}`}>
      <Typography.Text
        type={connected && agentOnline ? 'success' : 'secondary'}
        className="servers-console-status"
      >
        {connected && agentOnline ? t('console.connected') : t('console.connecting')}
      </Typography.Text>
      <pre ref={preRef} className="servers-console-output">
        {lines.join('\n')}
      </pre>
      <Space.Compact className="servers-console-input">
        <Input
          placeholder={t('console.commandPlaceholder')}
          value={command}
          disabled={!connected || !agentOnline}
          onChange={(e) => setCommand(e.target.value)}
          onPressEnter={send}
        />
        <Button type="primary" disabled={!connected || !agentOnline} onClick={send}>
          {t('console.send')}
        </Button>
      </Space.Compact>
    </div>
  );
}

/** Stop/Restart only when Minecraft process is actually running. */
export function shouldShowMinecraftControls(server: { minecraft_running?: boolean }): boolean {
  return server.minecraft_running === true;
}

/** Console inside a game server card while provisioning or running. */
export function shouldShowGameServerConsole(
  game: { status: string },
  agentOnline: boolean,
): boolean {
  return (
    agentOnline &&
    (game.status === 'running' ||
      game.status === 'starting' ||
      game.status === 'installing' ||
      game.status === 'error')
  );
}

/** @deprecated Use shouldShowGameServerConsole for per-game consoles. */
export function shouldShowServerConsole(server: { minecraft_running?: boolean }): boolean {
  return server.minecraft_running === true;
}
