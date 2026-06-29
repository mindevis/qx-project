import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Popconfirm,
  Space,
  Spin,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  BuildOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { api, type GameServer } from '@/api/client';
import { ServerConsolePanel, shouldShowGameServerConsole } from '@/components/ServerConsolePanel';
import { GameServerPropertiesPanel } from '@/components/GameServerPropertiesPanel';
import { GameServerModsPanel } from '@/components/GameServerModsPanel';
import { GameServerFilesPanel } from '@/components/GameServerFilesPanel';
import {
  gameServerTypeCapabilities,
  gameServerTypeLabelText,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import {
  formatGameServerLoaderVersionLabel,
  formatGameServerMcVersionLabel,
} from '@/lib/gameServerVersions';
import {
  isVpsGameServerProvisioning,
  listVpsGameServers,
  reinstallVpsGameServer,
  removeVpsGameServer,
  restartVpsGameServer,
  startVpsGameServer,
  stopVpsGameServer,
  type VpsGameServer,
} from '@/lib/vpsGameServers';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { logger } from '@/lib/logger';
import './ServersPage.css';

const { Title, Paragraph, Text } = Typography;

function gameServerStatusColor(status: VpsGameServer['status']): string {
  switch (status) {
    case 'running':
      return 'success';
    case 'starting':
    case 'installing':
      return 'processing';
    case 'stopped':
      return 'default';
    case 'error':
      return 'error';
    default:
      return 'default';
  }
}

function useGameServerStatusLabel() {
  const { t } = useI18n();
  return useCallback(
    (status: VpsGameServer['status']) => t(`servers.gameStatus.${status}`),
    [t],
  );
}

export function GameServerDetailPage() {
  const { t } = useI18n();
  const message = useMessage();
  const navigate = useNavigate();
  const gameStatusLabel = useGameServerStatusLabel();
  const { id: vpsId, gameServerId } = useParams<{ id: string; gameServerId: string }>();
  const [vps, setVps] = useState<GameServer | null>(null);
  const [game, setGame] = useState<VpsGameServer | null>(null);
  const [loading, setLoading] = useState(true);
  const [reinstalling, setReinstalling] = useState(false);
  const [powerAction, setPowerAction] = useState(false);

  const load = useCallback(async () => {
    if (!vpsId || !gameServerId) return;
    try {
      const [server, games] = await Promise.all([
        api.getServer(vpsId),
        listVpsGameServers(vpsId),
      ]);
      setVps(server);
      const found = games.find((item) => item.id === gameServerId) ?? null;
      setGame(found);
      if (!found) {
        message.error(t('gameServerDetail.notFound'));
        navigate(`/servers/${vpsId}`);
      }
    } catch (e) {
      logger.warn('failed to load game server detail', { error: String(e) });
      message.error(t('gameServerDetail.loadFailed'));
      navigate(vpsId ? `/servers/${vpsId}` : '/servers');
    } finally {
      setLoading(false);
    }
  }, [gameServerId, message, navigate, t, vpsId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!vps?.agent_online || !game) return undefined;
    const needsPoll =
      isVpsGameServerProvisioning(game.status) ||
      game.status === 'running' ||
      game.status === 'starting';
    if (!needsPoll) return undefined;
    const timer = window.setInterval(() => void load(), 3000);
    return () => window.clearInterval(timer);
  }, [game, load, vps?.agent_online]);

  const runPowerAction = async (action: 'start' | 'stop' | 'restart') => {
    if (!vpsId || !gameServerId) return;
    setPowerAction(true);
    try {
      const fn =
        action === 'start'
          ? startVpsGameServer
          : action === 'stop'
            ? stopVpsGameServer
            : restartVpsGameServer;
      const updated = await fn(vpsId, gameServerId);
      setGame(updated);
      message.success(
        t(
          action === 'start'
            ? 'servers.gameServerStartStarted'
            : action === 'stop'
              ? 'servers.gameServerStopStarted'
              : 'servers.gameServerRestartStarted',
        ),
      );
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setPowerAction(false);
    }
  };

  const onDelete = async () => {
    if (!vpsId || !gameServerId) return;
    try {
      await removeVpsGameServer(vpsId, gameServerId);
      message.success(t('servers.gameServerDeleted'));
      navigate(`/servers/${vpsId}`);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    }
  };

  const onReinstall = async () => {
    if (!vpsId || !gameServerId) return;
    setReinstalling(true);
    try {
      const updated = await reinstallVpsGameServer(vpsId, gameServerId);
      setGame(updated);
      message.success(t('servers.gameServerReinstallStarted'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setReinstalling(false);
    }
  };

  if (loading || !vps || !game || !vpsId) {
    return (
      <div className="servers-page">
        <div className="servers-loading">
          <Spin size="large" />
        </div>
      </div>
    );
  }

  const agentOnline = vps.agent_online;
  const rowBusy =
    isVpsGameServerProvisioning(game.status) || reinstalling || powerAction;
  const canStart = game.status === 'stopped' || game.status === 'error';
  const canStop = game.status === 'running' || game.status === 'starting';
  const canRestart =
    !isVpsGameServerProvisioning(game.status) &&
    game.status !== 'installing' &&
    (canStart || canStop);
  const showConsole = shouldShowGameServerConsole(game, agentOnline);
  const serverType = (game.server_type ?? 'vanilla') as VpsGameServerType;
  const caps = gameServerTypeCapabilities(serverType);

  const tabItems = [
    {
      key: 'console',
      label: t('gameServerDetail.tabConsole'),
      children: showConsole ? (
        <ServerConsolePanel serverId={vpsId} gameServerId={game.id} agentOnline={agentOnline} />
      ) : (
        <Paragraph type="secondary">{t('gameServerDetail.consoleUnavailable')}</Paragraph>
      ),
    },
    {
      key: 'settings',
      label: t('gameServerDetail.tabSettings'),
      children: (
        <GameServerPropertiesPanel
          vpsId={vpsId}
          gameServerId={game.id}
          agentOnline={agentOnline}
        />
      ),
    },
    ...(caps.mods
      ? [
          {
            key: 'mods',
            label: t('gameServerDetail.tabMods'),
            children: (
              <GameServerModsPanel
                vpsId={vpsId}
                gameServerId={game.id}
                agentOnline={agentOnline}
                supportsMods={caps.mods}
              />
            ),
          },
        ]
      : []),
    {
      key: 'files',
      label: t('gameServerDetail.tabFiles'),
      children: (
        <GameServerFilesPanel vpsId={vpsId} gameServerId={game.id} agentOnline={agentOnline} />
      ),
    },
  ];

  return (
    <div className="servers-page servers-page--detail">
      <section className="servers-hero servers-hero--detail">
        <div className="servers-hero-ambient" aria-hidden>
          <span className="servers-hero-blob servers-hero-blob--1" />
          <span className="servers-hero-blob servers-hero-blob--2" />
          <span className="servers-hero-grid-pattern" />
        </div>
        <div className="servers-hero-inner">
          <div className="servers-hero-content">
            <Link to={`/servers/${vpsId}`} className="servers-detail-back">
              <ArrowLeftOutlined /> {t('gameServerDetail.backToDedicated')}
            </Link>
            <span className="servers-badge">{t('gameServerDetail.badge')}</span>
            <Title level={1} className="servers-title">
              <span className="servers-title-highlight">{game.name}</span>
            </Title>
            <div className="servers-card-tags">
              <Tag color={gameServerStatusColor(game.status)}>{gameStatusLabel(game.status)}</Tag>
              <Tag>{gameServerTypeLabelText(t, serverType)}</Tag>
            </div>
            <Paragraph className="servers-intro">
              {formatGameServerMcVersionLabel(game.mc_version)} ·{' '}
              {formatGameServerLoaderVersionLabel(game.loader_version, game.server_type)}
              {game.address ? ` · ${game.address}:${game.port ?? '—'}` : null}
            </Paragraph>
          </div>
        </div>
      </section>

      <section className="servers-section">
        <div className="servers-panel">
          <div className="servers-game-card-actions game-server-detail-actions">
            <Button
              type="default"
              icon={canStop ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              loading={powerAction}
              disabled={rowBusy || !agentOnline || (!canStart && !canStop)}
              onClick={() => {
                if (canStop) void runPowerAction('stop');
                else if (canStart) void runPowerAction('start');
              }}
            >
              {canStop ? t('servers.gameServerStop') : t('servers.gameServerStart')}
            </Button>
            <Button
              icon={<SyncOutlined />}
              disabled={rowBusy || !agentOnline || !canRestart}
              onClick={() => void runPowerAction('restart')}
            >
              {t('servers.gameServerRestart')}
            </Button>
            <Popconfirm
              title={t('servers.reinstallGameServerConfirm')}
              disabled={rowBusy || !agentOnline}
              onConfirm={() => void onReinstall()}
            >
              <Button icon={<BuildOutlined />} loading={reinstalling} disabled={rowBusy || !agentOnline}>
                {t('servers.reinstallGameServer')}
              </Button>
            </Popconfirm>
            <Popconfirm title={t('servers.deleteGameServerConfirm')} onConfirm={() => void onDelete()}>
              <Button danger icon={<DeleteOutlined />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          </div>

          {game.rcon_port != null && game.rcon_port > 0 ? (
            <Space className="game-server-detail-rcon" wrap>
              <Text type="secondary">{t('servers.gameServerRconPort')}: {game.rcon_port}</Text>
              {game.rcon_password ? (
                <Text copyable={{ text: game.rcon_password }} code>
                  {t('servers.gameServerRconPassword')}: {game.rcon_password}
                </Text>
              ) : null}
            </Space>
          ) : null}

          <Tabs className="game-server-detail-tabs" items={tabItems} />
        </div>
      </section>
    </div>
  );
}
