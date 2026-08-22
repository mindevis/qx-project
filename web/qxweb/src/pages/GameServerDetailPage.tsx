import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  Alert,
  Breadcrumb,
  Button,
  Descriptions,
  Popconfirm,
  Space,
  Spin,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import {
  AppstoreOutlined,
  BuildOutlined,
  CodeOutlined,
  CopyOutlined,
  DeleteOutlined,
  FolderOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  SettingOutlined,
  SyncOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { api, type GameServer } from '@/api/client';
import { ServerConsolePanel, shouldShowGameServerConsole } from '@/components/ServerConsolePanel';
import { GameServerLaunchSettingsPanel } from '@/components/GameServerLaunchSettingsPanel';
import { GameServerPropertiesPanel } from '@/components/GameServerPropertiesPanel';
import { GameServerContentPanel } from '@/components/GameServerContentPanel';
import { GameServerModsPanel } from '@/components/GameServerModsPanel';
import { GameServerFilesPanel } from '@/components/GameServerFilesPanel';
import { GameServerModConfigsPanel } from '@/components/GameServerModConfigsPanel';
import { GameServerInstanceBinding } from '@/components/GameServerInstanceBinding';
import {
  gameServerTypeCapabilities,
  gameServerTypeLabelText,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import {
  formatGameServerMcVersionLabel,
} from '@/lib/gameServerVersions';
import {
  isVpsGameServerProvisioning,
  listVpsGameServers,
  cloneVpsGameServer,
  reinstallVpsGameServer,
  removeVpsGameServer,
  restartVpsGameServer,
  startVpsGameServer,
  stopVpsGameServer,
  type VpsGameServer,
} from '@/lib/vpsGameServers';
import { useI18n } from '@/i18n/I18nContext';
import { getAgentConnectionStatusKey } from '@/i18n';
import { getAgentConnectionStatus } from '@/lib/agentStatus';
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
  const [searchParams, setSearchParams] = useSearchParams();
  const [showRconPassword, setShowRconPassword] = useState(false);
  const [vps, setVps] = useState<GameServer | null>(null);
  const [game, setGame] = useState<VpsGameServer | null>(null);
  const [loading, setLoading] = useState(true);
  const [reinstalling, setReinstalling] = useState(false);
  const [cloning, setCloning] = useState(false);
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
      game.status === 'starting' ||
      game.status === 'error';
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

  const onClone = async () => {
    if (!vpsId || !gameServerId) return;
    setCloning(true);
    try {
      const cloned = await cloneVpsGameServer(vpsId, gameServerId);
      message.success(t('servers.gameServerCloned'));
      navigate(`/servers/${vpsId}/game-servers/${cloned.id}`);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setCloning(false);
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
    isVpsGameServerProvisioning(game.status) || reinstalling || powerAction || cloning;
  const canStart = game.status === 'stopped' || game.status === 'error';
  const canStop = game.status === 'running' || game.status === 'starting';
  const canRestart =
    !isVpsGameServerProvisioning(game.status) &&
    game.status !== 'installing' &&
    (canStart || canStop);
  const showConsole = shouldShowGameServerConsole(game, agentOnline);
  const serverType = (game.server_type ?? 'vanilla') as VpsGameServerType;
  const caps = gameServerTypeCapabilities(serverType);
  const agentConnection = getAgentConnectionStatus(vps);
  const agentConnectionKey = getAgentConnectionStatusKey(agentConnection);
  const agentConnectionLabel = t(agentConnectionKey);

  const tabLabel = (icon: ReactNode, label: string) => (
    <span className="game-server-detail-tab-label">
      {icon}
      {label}
    </span>
  );

  const tabItems = [
    {
      key: 'console',
      label: tabLabel(<CodeOutlined aria-hidden />, t('gameServerDetail.tabConsole')),
      children: showConsole ? (
        <ServerConsolePanel serverId={vpsId} gameServerId={game.id} agentOnline={agentOnline} />
      ) : (
        <Paragraph type="secondary">{t('gameServerDetail.consoleUnavailable')}</Paragraph>
      ),
    },
    {
      key: 'settings',
      label: tabLabel(<SettingOutlined aria-hidden />, t('gameServerDetail.tabSettings')),
      children: (
        <div className="game-server-settings-tab">
          <GameServerLaunchSettingsPanel
            vpsId={vpsId}
            game={game}
            disabled={rowBusy}
            onUpdated={setGame}
          />
          <section className="game-server-properties-section">
            <Title level={4} className="game-server-launch-title">
              {t('gameServerDetail.propertiesTitle')}
            </Title>
            <GameServerPropertiesPanel
              vpsId={vpsId}
              gameServerId={game.id}
              agentOnline={agentOnline}
            />
          </section>
        </div>
      ),
    },
    ...(caps.mods
      ? [
          {
            key: 'mods',
            label: tabLabel(<AppstoreOutlined aria-hidden />, t('gameServerDetail.tabMods')),
            children: (
              <GameServerModsPanel
                vpsId={vpsId}
                gameServerId={game.id}
                agentOnline={agentOnline}
                supportsMods={caps.mods}
                serverType={serverType}
                mcVersion={game.mc_version ?? '1.21'}
              />
            ),
          },
          {
            key: 'mod-configs',
            label: tabLabel(<CodeOutlined aria-hidden />, t('gameServerDetail.tabModConfigs')),
            children: (
              <GameServerModConfigsPanel
                vpsId={vpsId}
                gameServerId={game.id}
                agentOnline={agentOnline}
                mcVersion={game.mc_version ?? '1.21'}
                loader={serverType}
              />
            ),
          },
        ]
      : []),
    ...(!caps.mods && caps.clientContent
      ? [
          {
            key: 'resources',
            label: tabLabel(<AppstoreOutlined aria-hidden />, t('gameServerDetail.tabResources')),
            children: (
              <GameServerModsPanel
                vpsId={vpsId}
                gameServerId={game.id}
                agentOnline={agentOnline}
                supportsMods={false}
                serverType={serverType}
                mcVersion={game.mc_version ?? '1.21'}
              />
            ),
          },
        ]
      : []),
    ...(caps.plugins
      ? [
          {
            key: 'plugins',
            label: tabLabel(<ThunderboltOutlined aria-hidden />, t('gameServerDetail.tabPlugins')),
            children: (
              <GameServerContentPanel
                kind="plugin"
                vpsId={vpsId}
                gameServerId={game.id}
                agentOnline={agentOnline}
                supported={caps.plugins}
                serverType={serverType}
                mcVersion={game.mc_version ?? '1.21'}
              />
            ),
          },
        ]
      : []),
    {
      key: 'files',
      label: tabLabel(<FolderOutlined aria-hidden />, t('gameServerDetail.tabFiles')),
      children: (
        <GameServerFilesPanel vpsId={vpsId} gameServerId={game.id} agentOnline={agentOnline} />
      ),
    },
  ];

  return (
    <div className="servers-page servers-page--detail">
      <section className="servers-hero servers-hero--detail game-server-detail-hero">
        <div className="servers-hero-ambient" aria-hidden>
          <span className="servers-hero-blob servers-hero-blob--1" />
          <span className="servers-hero-blob servers-hero-blob--2" />
          <span className="servers-hero-grid-pattern" />
        </div>

        <div className="servers-hero-inner">
          <div className="servers-hero-content">
            <Breadcrumb
              className="game-server-detail-breadcrumb"
              items={[
                {
                  title: <Link to="/servers">{t('layout.navServers')}</Link>,
                },
                {
                  title: <Link to={`/servers/${vpsId}`}>{vps.name}</Link>,
                },
                {
                  title: game.name,
                },
              ]}
            />
            <span className="servers-badge">{t('gameServerDetail.badge')}</span>
            <Title level={1} className="servers-title">
              <span className="servers-title-highlight">{game.name}</span>
            </Title>
          </div>
        </div>
      </section>

      <section className="servers-section">
        <div className="servers-panel">
          <Descriptions
            className="game-server-detail-summary"
            size="small"
            column={{ xs: 1, sm: 2, md: 3 }}
            items={[
              {
                key: 'status',
                label: t('servers.gameServerStatus'),
                children: (
                  <Tag color={gameServerStatusColor(game.status)}>{gameStatusLabel(game.status)}</Tag>
                ),
              },
              {
                key: 'mc',
                label: t('servers.gameServerMcVersion'),
                children: formatGameServerMcVersionLabel(game.mc_version),
              },
              {
                key: 'type',
                label: t('servers.gameServerTypeLabel'),
                children: gameServerTypeLabelText(t, serverType),
              },
              {
                key: 'address',
                label: t('servers.gameServerPort'),
                children: game.address ? (
                  <Text copyable>{`${game.address}:${game.port ?? '—'}`}</Text>
                ) : (
                  '—'
                ),
              },
              {
                key: 'agent',
                label: t('gameServerDetail.summaryAgent'),
                children: agentConnectionLabel,
              },
            ]}
          />

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
            <Popconfirm
              title={t('servers.cloneGameServerConfirm')}
              disabled={rowBusy || !agentOnline}
              onConfirm={() => void onClone()}
            >
              <Button
                icon={<CopyOutlined />}
                loading={cloning}
                disabled={rowBusy || !agentOnline}
              >
                {t('servers.cloneGameServer')}
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
                <Space size={8}>
                  <Text copyable={{ text: game.rcon_password }} code>
                    {t('servers.gameServerRconPassword')}:{' '}
                    {showRconPassword ? game.rcon_password : '••••••••'}
                  </Text>
                  <Button type="link" size="small" onClick={() => setShowRconPassword((open) => !open)}>
                    {showRconPassword ? t('common.hide') : t('common.show')}
                  </Button>
                </Space>
              ) : null}
            </Space>
          ) : null}

          {game.status === 'error' && game.last_error ? (
            <Alert
              className="game-server-detail-crash"
              type="error"
              showIcon
              title={t('gameServerDetail.crashTitle')}
              description={
                <pre className="game-server-detail-crash-log">{game.last_error}</pre>
              }
            />
          ) : null}

          <GameServerInstanceBinding
            gameServerId={game.id}
            hasAddress={Boolean(game.address?.trim())}
          />

          <Tabs
            className="game-server-detail-tabs"
            activeKey={
              tabItems.some((item) => item.key === searchParams.get('tab'))
                ? searchParams.get('tab')!
                : showConsole
                  ? 'console'
                  : 'settings'
            }
            destroyOnHidden
            onChange={(key) => {
              setSearchParams(
                (prev) => {
                  const next = new URLSearchParams(prev);
                  next.set('tab', key);
                  return next;
                },
                { replace: true },
              );
            }}
            items={tabItems}
          />
        </div>
      </section>
    </div>
  );
}
