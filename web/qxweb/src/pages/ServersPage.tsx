import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent, type ReactNode } from 'react';
import { Link, Navigate, Route, Routes, useNavigate, useParams } from 'react-router-dom';
import {
  Alert,
  Button,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  CloudServerOutlined,
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  LoginOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
  RocketOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { api, type GameServer } from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { modalMotionProps } from '@/lib/modal';
import { useAuthModal } from '@/auth/AuthModalContext';
import { getDedicatedHostStatusKey, getAgentDeployStatusKey, getAgentConnectionStatusKey } from '@/i18n';
import {
  getAgentConnectionStatus,
  getAgentDeployStatus,
  isAgentDeployed,
  agentConnectionStatusColor,
  agentDeployStatusColor,
  type AgentConnectionStatus,
  type AgentDeployStatus,
} from '@/lib/agentStatus';
import {
  getVpsHostStatus,
  vpsHostStatusColor,
  type VpsHostStatus,
} from '@/lib/vpsStatus';
import { useI18n } from '@/i18n/I18nContext';
import { logger } from '@/lib/logger';
import {
  isVpsGameServerProvisioning,
  listVpsGameServers,
  restartVpsGameServer,
  removeVpsGameServer,
  startVpsGameServer,
  stopVpsGameServer,
  type VpsGameServer,
} from '@/lib/vpsGameServers';
import { useMessage } from '@/hooks/useMessage';
import {
  gameServerTypeLabelText,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import {
  formatGameServerLoaderVersionLabel,
  formatGameServerMcVersionLabel,
} from '@/lib/gameServerVersions';
import { AddGameServerModal } from '@/components/servers/AddGameServerModal';
import { EditGameServerModal } from '@/components/servers/EditGameServerModal';
import './ServersPage.css';
import { GameServerDetailPage } from './GameServerDetailPage';

const { TextArea } = Input;
const { Title, Paragraph, Text } = Typography;

function useStatusLabels() {
  const { t } = useI18n();
  const label = (key: string, fallback: string) => {
    const msg = t(key);
    return msg === key ? fallback : msg;
  };
  return {
    host: (status: VpsHostStatus) => label(getDedicatedHostStatusKey(status), status),
    agentDeploy: (status: AgentDeployStatus) => label(getAgentDeployStatusKey(status), status),
    agentConnection: (status: AgentConnectionStatus) =>
      label(getAgentConnectionStatusKey(status), status),
  };
}

function agentListTag(
  server: GameServer,
  labels: ReturnType<typeof useStatusLabels>,
): { text: string; color: string } {
  const deploy = getAgentDeployStatus(server);
  if (deploy === 'not_deployed') {
    return { text: labels.agentDeploy(deploy), color: agentDeployStatusColor(deploy) };
  }
  if (deploy === 'deploying') {
    return { text: labels.agentDeploy(deploy), color: agentDeployStatusColor(deploy) };
  }
  const connection = getAgentConnectionStatus(server);
  return {
    text: labels.agentConnection(connection),
    color: agentConnectionStatusColor(connection),
  };
}

function formatSshEndpoint(server: GameServer): string {
  return `${server.ssh.username}@${server.ssh.host}:${server.ssh.port}`;
}

function formatServerTimestamp(value?: string): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toLocaleString();
}

function isSshUnreachableError(message: string): boolean {
  const lower = message.toLowerCase();
  return (
    lower.includes('connection refused') ||
    lower.includes('actively refused') ||
    lower.includes('no route to host') ||
    lower.includes('i/o timeout') ||
    lower.includes('network is unreachable')
  );
}

function detailHeroClass(server: GameServer): string {
  const vpsStatus = getVpsHostStatus(server);
  if (vpsStatus === 'error') return 'servers-detail-hero-icon--error';
  if (server.agent_online) return 'servers-detail-hero-icon--agent';
  if (vpsStatus === 'active') return 'servers-detail-hero-icon--online';
  return 'servers-detail-hero-icon--offline';
}

function ServerDetailStat({
  icon,
  label,
  value,
  tone = 'default',
}: {
  icon: ReactNode;
  label: string;
  value: string;
  tone?: 'default' | 'success' | 'warning' | 'error' | 'info';
}) {
  return (
    <div className={`servers-detail-stat servers-detail-stat--${tone}`}>
      <span className="servers-detail-stat-icon" aria-hidden>
        {icon}
      </span>
      <span className="servers-detail-stat-label">{label}</span>
      <span className="servers-detail-stat-value">{value}</span>
    </div>
  );
}

function AgentDetailStat({
  deployStatus,
  connectionStatus,
  deployLabel,
  connectionLabel,
  version,
}: {
  deployStatus: AgentDeployStatus;
  connectionStatus: AgentConnectionStatus;
  deployLabel: string;
  connectionLabel: string;
  version?: string;
}) {
  const { t } = useI18n();
  const versionLabel = version?.trim() ? version : t('servers.agentVersionUnknown');
  return (
    <div className="servers-detail-stat servers-detail-stat--agent">
      <span className="servers-detail-stat-icon" aria-hidden>
        <RocketOutlined />
      </span>
      <span className="servers-detail-stat-label">{t('servers.statAgent')}</span>
      <div className="servers-agent-stat-rows">
        <div className="servers-agent-stat-row">
          <span className="servers-agent-stat-key">{t('servers.statAgentDeploy')}</span>
          <Tag color={agentDeployStatusColor(deployStatus)} className="servers-agent-stat-tag">
            {deployLabel}
          </Tag>
        </div>
        <div className="servers-agent-stat-row">
          <span className="servers-agent-stat-key">{t('servers.statAgentConnection')}</span>
          <Tag color={agentConnectionStatusColor(connectionStatus)} className="servers-agent-stat-tag">
            {connectionLabel}
          </Tag>
        </div>
        <div className="servers-agent-stat-row">
          <span className="servers-agent-stat-key">{t('servers.statAgentVersion')}</span>
          <span className="servers-agent-stat-version">{versionLabel}</span>
        </div>
      </div>
    </div>
  );
}

function ServerCard({ server }: { server: GameServer }) {
  const { t } = useI18n();
  const labels = useStatusLabels();
  const vpsStatus = getVpsHostStatus(server);
  const agentTag = agentListTag(server, labels);

  return (
    <article className="servers-card">
      <div className="servers-card-head">
        <span className="servers-card-icon" aria-hidden>
          <CloudServerOutlined />
        </span>
        <div className="servers-card-body">
          <Title level={4} className="servers-card-title">
            {server.name}
          </Title>
          <div className="servers-card-tags">
            <Tag color={vpsHostStatusColor(vpsStatus)}>{labels.host(vpsStatus)}</Tag>
            <Tag color={agentTag.color}>{agentTag.text}</Tag>
          </div>
        </div>
      </div>
      <Paragraph className="servers-card-meta">{formatSshEndpoint(server)}</Paragraph>
      <div className="servers-card-footer">
        <span className="servers-card-type">{server.server_type}</span>
        <Link to={`/servers/${server.id}`}>
          <Button type="primary" ghost size="small">
            {t('common.open')}
          </Button>
        </Link>
      </div>
    </article>
  );
}

function CreateServerModal({
  open,
  creating,
  form,
  onCancel,
  onCreate,
}: {
  open: boolean;
  creating: boolean;
  form: ReturnType<typeof Form.useForm>[0];
  onCancel: () => void;
  onCreate: () => void;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const keyFileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ port: 22 });
    }
  }, [open, form]);

  const onKeyFileSelected = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    try {
      const text = await file.text();
      form.setFieldValue('private_key', text.trim());
      message.success(t('servers.sshKeyLoaded'));
    } catch {
      message.error(t('common.error'));
    }
  };

  return (
    <Modal
      title={t('servers.addDedicated')}
      open={open}
      onCancel={onCancel}
      onOk={() => void onCreate()}
      confirmLoading={creating}
      width={600}
      destroyOnHidden
      {...modalMotionProps}
    >
      <Paragraph className="servers-create-hint">{t('servers.createHint')}</Paragraph>
      <Form form={form} layout="vertical" initialValues={{ port: 22 }}>
        <Form.Item
          name="name"
          label={t('common.name')}
          rules={[{ required: true, message: t('servers.nameRequired') }]}
        >
          <Input placeholder="Survival" />
        </Form.Item>
        <Form.Item name="host" label={t('servers.sshHost')} rules={[{ required: true }]}>
          <Input placeholder="203.0.113.10" />
        </Form.Item>
        <Form.Item name="port" label={t('servers.sshPort')}>
          <InputNumber min={1} max={65535} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="username" label={t('servers.sshUser')} rules={[{ required: true }]}>
          <Input placeholder="root" />
        </Form.Item>
        <Form.Item
          name="private_key"
          label={t('servers.sshKey')}
          rules={[{ required: true, message: t('servers.sshKeyRequired') }]}
          extra={
            <Space orientation="vertical" size={4} className="servers-create-key-extra">
              <span>{t('servers.sshKeyUploadHint')}</span>
              <input
                ref={keyFileInputRef}
                type="file"
                accept=".pem,.key,.pub,.txt,text/plain,application/x-pem-file,application/octet-stream"
                className="servers-create-key-file-input"
                onChange={(event) => void onKeyFileSelected(event)}
              />
              <Button
                icon={<UploadOutlined />}
                onClick={() => keyFileInputRef.current?.click()}
              >
                {t('servers.sshKeyUpload')}
              </Button>
            </Space>
          }
        >
          <TextArea rows={4} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
        </Form.Item>
        <Form.Item
          name="private_key_passphrase"
          label={t('servers.sshKeyPassphrase')}
          extra={t('servers.sshKeyPassphraseHint')}
        >
          <Input.Password autoComplete="new-password" placeholder="••••••••" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function ServersList() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const message = useMessage();
  const [servers, setServers] = useState<GameServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm();

  const stats = useMemo(() => {
    const online = servers.filter((s) => s.agent_online).length;
    return { total: servers.length, online };
  }, [servers]);

  const load = useCallback(async () => {
    try {
      const res = await api.listServers();
      setServers(res.items ?? []);
    } catch (e) {
      logger.warn('failed to load servers', { error: String(e) });
      message.error(t('servers.loadFailed'));
    }
  }, [message, t]);

  useEffect(() => {
    void (async () => {
      setLoading(true);
      await load();
      setLoading(false);
    })();
  }, [load]);

  const refresh = async () => {
    setRefreshing(true);
    await load();
    setRefreshing(false);
    message.success(t('servers.refreshed'));
  };

  const openCreate = () => {
    setCreateOpen(true);
  };

  const onCreate = async () => {
    try {
      const values = await form.validateFields();
      setCreating(true);
      const server = await api.createServer({
        name: values.name,
        ssh: {
          host: values.host,
          port: values.port ?? 22,
          username: values.username,
          private_key: values.private_key,
          private_key_passphrase: values.private_key_passphrase?.trim() || undefined,
        },
      });
      message.success(t('servers.added'));
      setCreateOpen(false);
      form.resetFields();
      navigate(`/servers/${server.id}`);
    } catch (e) {
      if (e instanceof Error && e.message) {
        message.error(e.message);
      }
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="servers-page">
      <section className="servers-hero">
        <div className="servers-hero-ambient" aria-hidden>
          <span className="servers-hero-blob servers-hero-blob--1" />
          <span className="servers-hero-blob servers-hero-blob--2" />
          <span className="servers-hero-blob servers-hero-blob--3" />
          <span className="servers-hero-grid-pattern" />
        </div>

        <div className="servers-hero-inner">
          <div className="servers-hero-content">
            <span className="servers-badge">{t('servers.badge')}</span>
            <Title level={1} className="servers-title">
              <span className="servers-title-highlight">{t('servers.title')}</span>
            </Title>
            <Paragraph className="servers-intro">{t('servers.intro')}</Paragraph>
            <div className="servers-hero-actions">
              <Button type="primary" size="large" icon={<PlusOutlined />} onClick={openCreate}>
                {t('servers.addDedicated')}
              </Button>
            </div>
          </div>

          <div className="servers-hero-visual" aria-hidden>
            <div className="servers-orbit">
              <div className="servers-orbit-ring" />
              <div className="servers-orbit-ring servers-orbit-ring--inner" />
              <div className="servers-orbit-core">
                <CloudServerOutlined />
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="servers-section">
        <div className="servers-section-header">
          <div>
            <span className="servers-section-eyebrow">{t('servers.workspaceEyebrow')}</span>
            <Title level={2} className="servers-section-title">
              {t('servers.workspaceTitle')}
            </Title>
          </div>
          <div className="servers-section-actions">
            <Button icon={<ReloadOutlined />} loading={refreshing} onClick={() => void refresh()}>
              {t('servers.refresh')}
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              {t('servers.addDedicated')}
            </Button>
          </div>
        </div>

        {loading ? (
          <div className="servers-loading">
            <Spin size="large" />
          </div>
        ) : (
          <>
            {servers.length > 0 ? (
              <div className="servers-stats">
                <div className="servers-stat">
                  <span className="servers-stat-value">{stats.total}</span>
                  <span className="servers-stat-label">{t('servers.statTotal')}</span>
                </div>
                <div className="servers-stat servers-stat--online">
                  <span className="servers-stat-value">{stats.online}</span>
                  <span className="servers-stat-label">{t('servers.statOnline')}</span>
                </div>
              </div>
            ) : null}

            {servers.length === 0 ? (
              <div className="servers-empty">
                <CloudServerOutlined className="servers-empty-icon" />
                <Title level={3} className="servers-empty-title">
                  {t('servers.emptyTitle')}
                </Title>
                <Paragraph className="servers-empty-desc">{t('servers.empty')}</Paragraph>
                <Button type="primary" size="large" icon={<PlusOutlined />} onClick={openCreate}>
                  {t('servers.addDedicated')}
                </Button>
              </div>
            ) : (
              <div className="servers-grid">
                {servers.map((item) => (
                  <ServerCard key={item.id} server={item} />
                ))}
              </div>
            )}
          </>
        )}
      </section>

      <CreateServerModal
        open={createOpen}
        creating={creating}
        form={form}
        onCancel={() => setCreateOpen(false)}
        onCreate={onCreate}
      />
    </div>
  );
}

function gameServerStatusColor(status: VpsGameServer['status']): string {
  switch (status) {
    case 'running':
      return 'success';
    case 'installing':
    case 'starting':
      return 'processing';
    case 'error':
      return 'error';
    default:
      return 'default';
  }
}

function useGameServerStatusLabel() {
  const { t } = useI18n();
  return (status: VpsGameServer['status']) => t(`servers.gameStatus.${status}`);
}

function GameServersCardList({
  vpsId,
  games,
  agentOnline,
  powerActionId,
  onDelete,
  onEdit,
  onStart,
  onStop,
  onRestart,
  gameServerTypeLabel,
  gameStatusLabel,
}: {
  vpsId: string;
  games: VpsGameServer[];
  agentOnline: boolean;
  powerActionId: string | null;
  onDelete: (id: string) => void;
  onEdit: (game: VpsGameServer) => void;
  onStart: (id: string) => void;
  onStop: (id: string) => void;
  onRestart: (id: string) => void;
  gameServerTypeLabel: (type: VpsGameServerType | undefined) => string;
  gameStatusLabel: (status: VpsGameServer['status']) => string;
}) {
  const { t } = useI18n();
  const navigate = useNavigate();

  return (
    <div className="servers-game-list">
      {games.map((game) => {
        const rowBusy = isVpsGameServerProvisioning(game.status) || powerActionId === game.id;
        const canStart = game.status === 'stopped' || game.status === 'error';
        const canStop = game.status === 'running' || game.status === 'starting';
        const canRestart =
          !isVpsGameServerProvisioning(game.status) &&
          game.status !== 'installing' &&
          (canStart || canStop);
        const serverType = game.server_type as VpsGameServerType | undefined;

        return (
          <article
            key={game.id}
            className="servers-game-card servers-game-card--interactive"
            onClick={() => navigate(`/servers/${vpsId}/game-servers/${game.id}`)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                navigate(`/servers/${vpsId}/game-servers/${game.id}`);
              }
            }}
            role="link"
            tabIndex={0}
          >
            <div className="servers-game-card-header">
              <div className="servers-game-card-headline">
                <Link
                  to={`/servers/${vpsId}/game-servers/${game.id}`}
                  className="servers-game-card-title-link"
                  onClick={(e) => e.stopPropagation()}
                >
                  <Title level={5} className="servers-game-card-title">
                    {game.name}
                  </Title>
                </Link>
                <Tag color={gameServerStatusColor(game.status)} className="servers-game-card-status">
                  {gameStatusLabel(game.status)}
                </Tag>
              </div>
              <div
                className="servers-game-card-actions"
                onClick={(e) => e.stopPropagation()}
                onKeyDown={(e) => e.stopPropagation()}
              >
                <Button
                  type="text"
                  size="small"
                  icon={canStop ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                  loading={powerActionId === game.id}
                  disabled={rowBusy || !agentOnline || (!canStart && !canStop)}
                  onClick={() => {
                    if (canStop) onStop(game.id);
                    else if (canStart) onStart(game.id);
                  }}
                />
                <Button
                  type="text"
                  size="small"
                  icon={<SyncOutlined />}
                  disabled={rowBusy || !agentOnline || !canRestart}
                  onClick={() => onRestart(game.id)}
                />
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  disabled={rowBusy}
                  onClick={() => onEdit(game)}
                />
                <Popconfirm
                  title={t('servers.deleteGameServerConfirm')}
                  onConfirm={() => onDelete(game.id)}
                >
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </div>
            </div>

            <dl className="servers-game-card-meta servers-game-card-meta--grid">
              <div className="servers-game-card-meta-item">
                <dt>{t('servers.gameServerMcVersion')}</dt>
                <dd>{formatGameServerMcVersionLabel(game.mc_version)}</dd>
              </div>
              <div className="servers-game-card-meta-item">
                <dt>{t('servers.gameServerTypeLabel')}</dt>
                <dd>{gameServerTypeLabel(serverType)}</dd>
              </div>
              <div className="servers-game-card-meta-item">
                <dt>{t('servers.gameServerCoreVersion')}</dt>
                <dd>{formatGameServerLoaderVersionLabel(game.loader_version, game.server_type)}</dd>
              </div>
              <div className="servers-game-card-meta-item">
                <dt>{t('servers.gameServerPort')}</dt>
                <dd className="servers-game-card-port">
                  <span>{game.address ?? '—'}</span>
                  <span className="servers-game-card-port-sep">:</span>
                  <span>{game.port ?? '—'}</span>
                </dd>
              </div>
            </dl>
          </article>
        );
      })}
    </div>
  );
}

function VpsGameServersSection({
  vpsId,
  agentOnline,
  defaultAddress,
}: {
  vpsId: string;
  agentOnline: boolean;
  defaultAddress: string;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const gameStatusLabel = useGameServerStatusLabel();
  const [games, setGames] = useState<VpsGameServer[]>([]);
  const [gamesLoading, setGamesLoading] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const [editingGame, setEditingGame] = useState<VpsGameServer | null>(null);
  const [powerActionId, setPowerActionId] = useState<string | null>(null);

  const gameServerTypeLabel = useCallback(
    (type: VpsGameServerType | undefined) => gameServerTypeLabelText(t, type),
    [t],
  );

  const refresh = useCallback(async () => {
    try {
      const items = await listVpsGameServers(vpsId);
      setGames(items);
    } catch (e) {
      logger.warn('failed to load game servers', { error: String(e) });
    } finally {
      setGamesLoading(false);
    }
  }, [vpsId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!agentOnline) return undefined;
    const needsPoll = games.some(
      (item) =>
        isVpsGameServerProvisioning(item.status) ||
        item.status === 'running' ||
        item.status === 'starting',
    );
    if (!needsPoll) return undefined;
    const timer = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(timer);
  }, [agentOnline, games, refresh]);

  const runPowerAction = useCallback(
    async (gameServerId: string, action: 'start' | 'stop' | 'restart') => {
      setPowerActionId(gameServerId);
      try {
        if (action === 'start') {
          await startVpsGameServer(vpsId, gameServerId);
          message.success(t('servers.gameServerStartStarted'));
        } else if (action === 'stop') {
          await stopVpsGameServer(vpsId, gameServerId);
          message.success(t('servers.gameServerStopStarted'));
        } else {
          await restartVpsGameServer(vpsId, gameServerId);
          message.success(t('servers.gameServerRestartStarted'));
        }
        await refresh();
      } catch (e) {
        message.error(e instanceof Error ? e.message : t('common.error'));
      } finally {
        setPowerActionId(null);
      }
    },
    [message, refresh, t, vpsId],
  );

  const handleDelete = useCallback(
    (gameServerId: string) => {
      void removeVpsGameServer(vpsId, gameServerId)
        .then(() => refresh())
        .then(() => message.success(t('servers.gameServerDeleted')))
        .catch((e) => message.error(e instanceof Error ? e.message : t('common.error')));
    },
    [message, refresh, t, vpsId],
  );

  return (
    <div className="servers-panel">
      <div className="servers-panel-header">
        <Title level={4} className="servers-panel-title">
          {t('servers.gameServersTitle')}
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          disabled={!agentOnline}
          onClick={() => setAddOpen(true)}
        >
          {t('servers.addGameServer')}
        </Button>
      </div>
      {!agentOnline ? (
        <Paragraph type="secondary" className="servers-hint">
          {t('servers.gameServersAgentRequired')}
        </Paragraph>
      ) : gamesLoading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : games.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          className="servers-game-empty"
          description={
            <div className="servers-game-empty-copy">
              <Text strong className="servers-game-empty-title">
                {t('servers.noGameServers')}
              </Text>
              <Paragraph type="secondary" className="servers-game-empty-hint">
                {t('servers.noGameServersHint')}
              </Paragraph>
            </div>
          }
        />
      ) : (
        <GameServersCardList
          vpsId={vpsId}
          games={games}
          agentOnline={agentOnline}
          powerActionId={powerActionId}
          onDelete={handleDelete}
          onEdit={setEditingGame}
          /* v8 ignore next -- @preserve power actions covered in GameServerDetailPage tests */
          onStart={(id) => void runPowerAction(id, 'start')}
          /* v8 ignore next -- @preserve */
          onStop={(id) => void runPowerAction(id, 'stop')}
          /* v8 ignore next -- @preserve */
          onRestart={(id) => void runPowerAction(id, 'restart')}
          gameServerTypeLabel={gameServerTypeLabel}
          gameStatusLabel={gameStatusLabel}
        />
      )}

      <AddGameServerModal
        open={addOpen}
        vpsId={vpsId}
        defaultAddress={defaultAddress}
        existingGames={games}
        onClose={() => setAddOpen(false)}
        onCreated={() => void refresh()}
      />
      <EditGameServerModal
        open={editingGame != null}
        vpsId={vpsId}
        game={editingGame}
        existingGames={games}
        onClose={() => setEditingGame(null)}
        onUpdated={(updated) => {
          setGames((prev) => prev.map((item) => (item.id === updated.id ? updated : item)));
        }}
      />
    </div>
  );
}

function ServerDetail() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const message = useMessage();
  const labels = useStatusLabels();
  const [server, setServer] = useState<GameServer | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [action, setAction] = useState<string | null>(null);

  const load = useCallback(
    async (quiet = false) => {
      /* v8 ignore next 3 -- @preserve */
      if (!id) return;
      if (quiet) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      try {
        const data = await api.getServer(id);
        setServer(data);
      } catch (e) {
        logger.warn('failed to load server', { error: String(e) });
        if (!quiet) {
          message.error(t('servers.notFound'));
          navigate('/servers');
        }
      } finally {
        if (quiet) {
          setRefreshing(false);
        } else {
          setLoading(false);
        }
      }
    },
    [id, message, navigate, t],
  );

  useEffect(() => {
    void load(false);
    const timer = window.setInterval(() => void load(true), 5000);
    return () => window.clearInterval(timer);
  }, [load]);

  /* v8 ignore start -- @preserve clipboard and manual refresh are browser-integration paths */
  const handleRefresh = async () => {
    await load(true);
    message.success(t('servers.detailRefreshed'));
  };

  const copySsh = async () => {
    if (!server) return;
    try {
      await navigator.clipboard.writeText(formatSshEndpoint(server));
      message.success(t('servers.sshCopied'));
    } catch {
      message.error(t('servers.copyFailed'));
    }
  };

  const onCopySshClick = () => void copySsh();
  const onDetailRefreshClick = () => void handleRefresh();
  /* v8 ignore end */

  const runAgentAction = async (kind: 'deploy' | 'update') => {
    /* v8 ignore next 3 -- @preserve */
    if (!id) return;
    setAction(kind);
    try {
      const updated = await api.deployServer(id);
      setServer(updated);
      message.success(t(kind === 'deploy' ? 'servers.deployDone' : 'servers.updateAgentDone'));
    } catch (e) {
      const raw = e instanceof Error ? e.message : t('common.error');
      const text = isSshUnreachableError(raw) ? t('servers.deploySshUnreachable') : raw;
      message.error(text);
    } finally {
      setAction(null);
    }
  };

  const onDelete = async () => {
    /* v8 ignore next 3 -- @preserve */
    if (!id) return;
    try {
      await api.deleteServer(id);
      message.success(t('servers.deleted'));
      navigate('/servers');
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    }
  };

  if (loading || !server) {
    return (
      <div className="servers-page">
        <div className="servers-loading">
          <Spin size="large" />
        </div>
      </div>
    );
  }

  const busy = action !== null;
  const lastSeen = formatServerTimestamp(server.last_seen_at);
  const createdAt = formatServerTimestamp(server.created_at);
  const vpsStatus = getVpsHostStatus(server);
  const agentDeploy = getAgentDeployStatus(server);
  const agentConnection = getAgentConnectionStatus(server);
  const vpsTone =
    vpsStatus === 'error' ? 'error' : vpsStatus === 'active' ? 'success' : 'default';
  const agentTag = agentListTag(server, labels);

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
            <Link to="/servers" className="servers-detail-back">
              <ArrowLeftOutlined /> {t('servers.backToList')}
            </Link>
            <span className="servers-badge">{t('servers.detailBadge')}</span>
            <Title level={1} className="servers-title">
              <span className="servers-title-highlight">{server.name}</span>
            </Title>
            <Paragraph className="servers-intro servers-detail-endpoint">
              <code>{formatSshEndpoint(server)}</code>
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                aria-label={t('servers.copySsh')}
                onClick={onCopySshClick}
              />
            </Paragraph>
            <div className="servers-card-tags">
              <Tag color={vpsHostStatusColor(vpsStatus)}>{labels.host(vpsStatus)}</Tag>
              <Tag color={agentTag.color}>{agentTag.text}</Tag>
            </div>
          </div>

          <div className="servers-hero-visual" aria-hidden>
            <div className={`servers-detail-hero-icon ${detailHeroClass(server)}`}>
              <CloudServerOutlined />
            </div>
          </div>
        </div>
      </section>

      <section className="servers-section">
        <div className="servers-section-header">
          <div>
            <span className="servers-section-eyebrow">{t('servers.detailEyebrow')}</span>
            <Title level={2} className="servers-section-title">
              {t('servers.statusOverview')}
            </Title>
          </div>
          <div className="servers-section-actions">
            <Button icon={<ReloadOutlined />} loading={refreshing} onClick={onDetailRefreshClick}>
              {t('servers.refresh')}
            </Button>
          </div>
        </div>

        {vpsStatus === 'error' ? (
          <Alert type="error" showIcon title={t('servers.errorBanner')} className="servers-detail-alert" />
        ) : null}

        <div className="servers-detail-stats">
          <ServerDetailStat
            icon={<CloudServerOutlined />}
            label={t('servers.statDedicated')}
            value={labels.host(vpsStatus)}
            tone={vpsTone}
          />
          <AgentDetailStat
            deployStatus={agentDeploy}
            connectionStatus={agentConnection}
            deployLabel={labels.agentDeploy(agentDeploy)}
            connectionLabel={labels.agentConnection(agentConnection)}
            version={server.agent_version}
          />
        </div>

        <div className="servers-detail-grid">
          <div className="servers-detail-main">
            {!server.agent_online || !server.agent_deployed ? (
              <div className="servers-panel servers-panel--workflow">
                <Title level={4} className="servers-panel-title">
                  {t('servers.workflowTitle')}
                </Title>
                <ol className="servers-workflow">
                  <li className="servers-workflow-step servers-workflow-step--active">
                    <span className="servers-workflow-number">1</span>
                    <span>{t('servers.workflowStep1')}</span>
                  </li>
                  <li className="servers-workflow-step">
                    <span className="servers-workflow-number">2</span>
                    <span>{t('servers.workflowStep2')}</span>
                  </li>
                  <li className="servers-workflow-step">
                    <span className="servers-workflow-number">3</span>
                    <span>{t('servers.workflowStep3')}</span>
                  </li>
                </ol>
              </div>
            ) : null}

            <div className="servers-panel">
              <Title level={4} className="servers-panel-title">
                {t('servers.infoTitle')}
              </Title>
              <div className="servers-info-rows">
                <div className="servers-info-row">
                  <span className="servers-info-label">{t('servers.sshLabel')}</span>
                  <Text className="servers-info-value">{formatSshEndpoint(server)}</Text>
                </div>
                {createdAt ? (
                  <div className="servers-info-row">
                    <span className="servers-info-label">{t('servers.createdAt')}</span>
                    <Text className="servers-info-value">{createdAt}</Text>
                  </div>
                ) : null}
                {lastSeen ? (
                  <div className="servers-info-row">
                    <span className="servers-info-label">{t('servers.lastSeen')}</span>
                    <Text className="servers-info-value">{lastSeen}</Text>
                  </div>
                ) : null}
                {server.config.jar_path ? (
                  <div className="servers-info-row">
                    <span className="servers-info-label">JAR</span>
                    <Text className="servers-info-value">{server.config.jar_path}</Text>
                  </div>
                ) : null}
                {server.config.jvm_args && server.config.jvm_args.length > 0 ? (
                  <div className="servers-info-row">
                    <span className="servers-info-label">JVM</span>
                    <Text className="servers-info-value">{server.config.jvm_args.join(' ')}</Text>
                  </div>
                ) : null}
              </div>
            </div>

            <VpsGameServersSection
              vpsId={server.id}
              agentOnline={server.agent_online}
              defaultAddress={server.ssh.host}
            />

            <div className="servers-panel">
              <Title level={4} className="servers-panel-title">
                {t('servers.management')}
              </Title>
              <div className="servers-action-groups">
                {!isAgentDeployed(server) ? (
                  <div className="servers-action-group">
                    <Text type="secondary" className="servers-action-group-label">
                      {t('servers.actionDeploy')}
                    </Text>
                    <div className="servers-actions">
                      <Button
                        type="primary"
                        icon={<RocketOutlined />}
                        loading={busy && action === 'deploy'}
                        onClick={() => void runAgentAction('deploy')}
                      >
                        {t('servers.deployAgent')}
                      </Button>
                    </div>
                    <Paragraph type="secondary" className="servers-hint">
                      {t('servers.deployHint')}
                    </Paragraph>
                  </div>
                ) : (
                  <div className="servers-action-group">
                    <Text type="secondary" className="servers-action-group-label">
                      {t('servers.actionAgent')}
                    </Text>
                    <div className="servers-actions">
                      <Button
                        type="primary"
                        icon={<ReloadOutlined />}
                        loading={busy && action === 'update'}
                        onClick={() => void runAgentAction('update')}
                      >
                        {t('servers.updateAgent')}
                      </Button>
                    </div>
                    <Paragraph type="secondary" className="servers-hint">
                      {t('servers.updateAgentHint')}
                    </Paragraph>
                  </div>
                )}

                <div className="servers-action-group servers-action-group--danger">
                  <Text type="secondary" className="servers-action-group-label">
                    {t('servers.dangerZone')}
                  </Text>
                  <Popconfirm title={t('servers.deleteConfirm')} onConfirm={() => void onDelete()}>
                    <Button danger icon={<DeleteOutlined />}>
                      {t('common.delete')}
                    </Button>
                  </Popconfirm>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}

function ServersAuthGate() {
  const { t } = useI18n();
  const { openAuthModal } = useAuthModal();

  return (
    <div className="servers-page">
      <section className="servers-auth-section">
        <LoginOutlined className="servers-auth-icon" />
        <Title level={3}>{t('servers.authRequiredTitle')}</Title>
        <Paragraph>{t('servers.authRequired')}</Paragraph>
        <Paragraph type="secondary">{t('servers.authRequiredDesc')}</Paragraph>
        <Button type="primary" size="large" icon={<LoginOutlined />} onClick={() => openAuthModal('login')}>
          {t('auth.signIn')}
        </Button>
      </section>
    </div>
  );
}

export function ServersPage() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div className="servers-page">
        <div className="servers-loading">
          <Spin size="large" />
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <ServersAuthGate />;
  }

  return (
    <Routes>
      <Route index element={<ServersList />} />
      <Route path=":id/game-servers/:gameServerId" element={<GameServerDetailPage />} />
      <Route path=":id" element={<ServerDetail />} />
      <Route path="*" element={<Navigate to="/servers" replace />} />
    </Routes>
  );
}
