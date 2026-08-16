import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Col,
  Empty,
  Input,
  Rate,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  CopyOutlined,
  CrownOutlined,
  HeartFilled,
  HeartOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import {
  api,
  type MojangLinkStatus,
  type MonitoringInstanceBinding,
  type MonitoringServer,
  type OfflineProfile,
} from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { ALL_GAME_SERVER_TYPES, gameServerTypeLabelText } from '@/lib/gameServerTypes';
import { isLaunchStarted } from '@/lib/launchProgress';
import { cachedListMcVersions } from '@/lib/mcVersionsCache';
import { offlineUsernameFromIdentity } from '@/lib/monitoringConnect';
import { isPrepareTerminal } from '@/lib/prepareProgress';
import type { ConnectProgressStep } from '@/lib/connectProgress';
import { ConnectClientModsModal } from '@/components/ConnectClientModsModal';
import { ConnectProgressModal } from '@/components/ConnectProgressModal';
import { highlightMinecraft } from '@/pages/HomePage';
import './MonitoringPage.css';

const { Title, Paragraph, Text } = Typography;

const LAUNCH_POLL_MS = 1500;
const LAUNCH_POLL_MAX = 120;
const PREPARE_POLL_MAX = 400;

type Filters = {
  mc_version: string;
  loader: string;
  mod: string;
  plugin: string;
  q: string;
};

const EMPTY_FILTERS: Filters = {
  mc_version: '',
  loader: '',
  mod: '',
  plugin: '',
  q: '',
};

type SortBy = 'online' | 'rating' | 'likes' | 'name';

type ConnectFlowState = {
  server: MonitoringServer;
  step: ConnectProgressStep;
  status?: string;
  detail?: string;
  errorCode?: string;
  failed?: boolean;
};

type PreparePollResult = {
  status: string;
  error_code?: string;
  progress_message?: string;
};

const SEARCH_DEBOUNCE_MS = 400;

function serverEndpoint(server: MonitoringServer): string {
  return `${server.address}:${server.port}`;
}

function MonitoringServerCard({
  server,
  liked,
  onLike,
  onRate,
  canInteract,
  onRequireAuth,
  loaderLabel,
  boundInstanceName,
  onConnect,
  connecting,
  isGuest,
}: {
  server: MonitoringServer;
  liked: boolean;
  onLike: (server: MonitoringServer) => void;
  onRate: (server: MonitoringServer, rating: number) => void;
  canInteract: boolean;
  onRequireAuth: () => void;
  loaderLabel: string;
  boundInstanceName?: string;
  onConnect: (server: MonitoringServer) => void;
  connecting: boolean;
  isGuest: boolean;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const endpoint = serverEndpoint(server);

  const copyAddress = async () => {
    try {
      await navigator.clipboard.writeText(endpoint);
      message.success(t('monitoring.copied'));
    } catch {
      message.error(t('common.error'));
    }
  };

  const handleLike = () => {
    if (!canInteract) {
      onRequireAuth();
      return;
    }
    onLike(server);
  };

  const handleRate = (value: number) => {
    if (!canInteract) {
      onRequireAuth();
      return;
    }
    onRate(server, value);
  };

  return (
    <article
      className={[
        'monitoring-card',
        server.is_premium ? 'monitoring-card--premium' : '',
        server.is_online ? 'monitoring-card--online' : 'monitoring-card--offline',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div
        className="monitoring-card-banner"
        style={
          server.banner_url
            ? { backgroundImage: `url(${server.banner_url})` }
            : undefined
        }
      >
        {server.is_premium ? (
          <span className="monitoring-card-premium-badge">
            <CrownOutlined aria-hidden />
            {t('monitoring.premium')}
          </span>
        ) : null}
        <span className="monitoring-card-likes-badge" title={t('monitoring.likesBadge')}>
          <TeamOutlined aria-hidden />
          {t('monitoring.likesCount', { count: server.likes_count })}
        </span>
        <span
          className={[
            'monitoring-card-status',
            server.is_online ? 'monitoring-card-status--online' : 'monitoring-card-status--offline',
          ].join(' ')}
        >
          <span className="monitoring-card-status-dot" aria-hidden />
          {server.is_online ? t('monitoring.online') : t('monitoring.offline')}
        </span>
      </div>

      <div className="monitoring-card-body">
        <div className="monitoring-card-head">
          <div className="monitoring-card-title-row">
            <span className="monitoring-card-online-dot" aria-hidden />
            <Title level={4} className="monitoring-card-title">
              {server.name}
            </Title>
          </div>
          <Space size={4} wrap className="monitoring-card-meta">
            <Tag>{server.mc_version}</Tag>
            <Tag color="blue">{loaderLabel}</Tag>
            {server.loader_version ? <Tag color="geekblue">{server.loader_version}</Tag> : null}
          </Space>
        </div>

        {server.description ? (
          <Paragraph className="monitoring-card-description" ellipsis={{ rows: 2 }}>
            {server.description}
          </Paragraph>
        ) : null}

        {server.tags.length > 0 ? (
          <Space size={[4, 4]} wrap className="monitoring-card-tags">
            {server.tags.map((tag) => (
              <Tag key={tag} className="monitoring-card-tag">
                {tag}
              </Tag>
            ))}
          </Space>
        ) : null}

        <div className="monitoring-card-stats">
          <button type="button" className="monitoring-card-like" onClick={handleLike}>
            {liked ? <HeartFilled /> : <HeartOutlined />}
            <span>{server.likes_count}</span>
          </button>
          <div className="monitoring-card-rating">
            <Rate
              allowHalf={false}
              value={server.rating_avg > 0 ? Math.round(server.rating_avg) : 0}
              onChange={handleRate}
            />
            <Text type="secondary" className="monitoring-card-rating-text">
              {server.rating_count > 0
                ? t('monitoring.ratingCount', {
                    avg: server.rating_avg.toFixed(1),
                    count: server.rating_count,
                  })
                : t('monitoring.noRatings')}
            </Text>
          </div>
        </div>

        {boundInstanceName ? (
          <div className="monitoring-card-binding">
            <Text type="secondary" className="monitoring-card-binding-label">
              {t('monitoring.boundInstance')}
            </Text>
            <Text className="monitoring-card-binding-value">{boundInstanceName}</Text>
            <Text type="secondary" className="monitoring-card-binding-hint">
              {t('monitoring.boundInstanceLocked')}
            </Text>
          </div>
        ) : null}

        <div className="monitoring-card-actions">
          <Text className="monitoring-card-address" copyable={false}>
            {endpoint}
          </Text>
          <Space wrap>
            <Tooltip title={t('monitoring.copyIp')}>
              <Button icon={<CopyOutlined />} onClick={() => void copyAddress()}>
                {t('monitoring.copyIp')}
              </Button>
            </Tooltip>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              loading={connecting}
              onClick={() => onConnect(server)}
            >
              {t('monitoring.connect')}
            </Button>
          </Space>
          {isGuest ? (
            <Text type="secondary" className="monitoring-connect-guest-hint">
              {t('monitoring.connectGuestHint')}
            </Text>
          ) : null}
        </div>
      </div>
    </article>
  );
}

export function MonitoringPage() {
  const { t } = useI18n();
  const message = useMessage();
  const { isAuthenticated, user } = useAuth();
  const { openAuthModal } = useAuthModal();
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [draftQuery, setDraftQuery] = useState('');
  const [sortBy, setSortBy] = useState<SortBy>('online');
  const [servers, setServers] = useState<MonitoringServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [likedIds, setLikedIds] = useState<Set<string>>(new Set());
  const [mcVersions, setMcVersions] = useState<string[]>([]);
  const [bindings, setBindings] = useState<Map<string, MonitoringInstanceBinding>>(new Map());
  const [linkedDevice, setLinkedDevice] = useState<{ device_id: string } | null>(null);
  const [profiles, setProfiles] = useState<OfflineProfile[]>([]);
  const [mojangStatus, setMojangStatus] = useState<MojangLinkStatus | null>(null);
  const [connectingServerId, setConnectingServerId] = useState<string | null>(null);
  const [connectFlow, setConnectFlow] = useState<ConnectFlowState | null>(null);
  const connectGenRef = useRef(0);
  const [refreshing, setRefreshing] = useState(false);

  const onlineCount = useMemo(() => servers.filter((server) => server.is_online).length, [servers]);

  const loaderLabel = useCallback(
    (type: string) => gameServerTypeLabelText(t, type),
    [t],
  );

  useEffect(() => {
    void cachedListMcVersions()
      .then((data) => {
        const versions = [...new Set((data.items ?? []).map((item) => item.id))].sort().reverse();
        setMcVersions(versions);
      })
      .catch(() => {
        /* optional filter source */
      });
  }, []);

  useEffect(() => {
    if (!isAuthenticated) {
      setBindings(new Map());
      setLinkedDevice(null);
      setProfiles([]);
      setMojangStatus(null);
      return;
    }
    void (async () => {
      try {
        const [bindingData, device, profileData, mojang] = await Promise.all([
          api.listMonitoringBindings(),
          api.myLauncherDevice().catch(() => null),
          api.listProfiles().catch(() => ({ items: [] as OfflineProfile[] })),
          api.mojangStatus().catch(() => null),
        ]);
        setBindings(
          new Map((bindingData.items ?? []).map((item) => [item.game_server_id, item])),
        );
        setLinkedDevice(device?.device_id ? { device_id: device.device_id } : null);
        setProfiles(profileData.items ?? []);
        setMojangStatus(mojang);
      } catch {
        /* optional workspace data */
      }
    })();
  }, [isAuthenticated]);

  const loadServers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listMonitoringServers({
        mc_version: filters.mc_version || undefined,
        loader: filters.loader || undefined,
        mod: filters.mod || undefined,
        plugin: filters.plugin || undefined,
        q: filters.q || undefined,
      });
      setServers(data.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
      setServers([]);
    } finally {
      setLoading(false);
    }
  }, [filters, message, t]);

  useEffect(() => {
    void loadServers();
  }, [loadServers]);

  useEffect(() => {
    const id = window.setTimeout(() => {
      const q = draftQuery.trim();
      setFilters((prev) => (prev.q === q ? prev : { ...prev, q }));
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [draftQuery]);

  const sortedServers = useMemo(() => {
    const list = [...servers];
    switch (sortBy) {
      case 'rating':
        return list.sort(
          (a, b) =>
            b.rating_avg - a.rating_avg ||
            b.rating_count - a.rating_count ||
            a.name.localeCompare(b.name),
        );
      case 'likes':
        return list.sort(
          (a, b) => b.likes_count - a.likes_count || a.name.localeCompare(b.name),
        );
      case 'name':
        return list.sort((a, b) => a.name.localeCompare(b.name));
      case 'online':
      default:
        return list.sort(
          (a, b) =>
            Number(b.is_online) - Number(a.is_online) ||
            b.likes_count - a.likes_count ||
            a.name.localeCompare(b.name),
        );
    }
  }, [servers, sortBy]);

  const sortOptions = useMemo(
    () => [
      { value: 'online' as const, label: t('monitoring.sortOnline') },
      { value: 'rating' as const, label: t('monitoring.sortRating') },
      { value: 'likes' as const, label: t('monitoring.sortLikes') },
      { value: 'name' as const, label: t('monitoring.sortName') },
    ],
    [t],
  );

  const modOptions = useMemo(() => {
    const values = new Set<string>();
    for (const server of servers) {
      for (const mod of server.mods) values.add(mod);
    }
    return [...values].sort((a, b) => a.localeCompare(b));
  }, [servers]);

  const pluginOptions = useMemo(() => {
    const values = new Set<string>();
    for (const server of servers) {
      for (const plugin of server.plugins) values.add(plugin);
    }
    return [...values].sort((a, b) => a.localeCompare(b));
  }, [servers]);

  const handleLike = async (server: MonitoringServer) => {
    try {
      const updated = await api.likeMonitoringServer(server.id);
      setServers((prev) => prev.map((item) => (item.id === updated.id ? updated : item)));
      setLikedIds((prev) => new Set(prev).add(updated.id));
      message.success(t('monitoring.liked'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    }
  };

  const handleRate = async (server: MonitoringServer, rating: number) => {
    try {
      const updated = await api.rateMonitoringServer(server.id, rating);
      setServers((prev) => prev.map((item) => (item.id === updated.id ? updated : item)));
      message.success(t('monitoring.rated'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    }
  };

  const openMinecraftLink = (server: MonitoringServer) => {
    window.location.href = `minecraft://${server.address}:${server.port}`;
  };

  const updateConnectFlow = (patch: Partial<ConnectFlowState>) => {
    setConnectFlow((prev) => (prev ? { ...prev, ...patch } : prev));
  };

  const closeConnectFlow = () => {
    connectGenRef.current += 1;
    setConnectFlow(null);
    setConnectingServerId(null);
  };

  const pollLaunchRequest = async (requestId: string, gen: number) => {
    for (let attempt = 0; attempt < LAUNCH_POLL_MAX; attempt += 1) {
      if (connectGenRef.current !== gen) {
        return { status: 'cancelled' };
      }
      const req = await api.getLaunchRequest(requestId);
      updateConnectFlow({
        status: req.status,
        detail: req.progress_message,
        errorCode: req.error_code,
      });
      if (isLaunchStarted(req.status)) {
        return req;
      }
      await new Promise((resolve) => window.setTimeout(resolve, LAUNCH_POLL_MS));
    }
    return { status: 'timeout' };
  };

  const pollPrepareRequest = async (requestId: string, gen: number): Promise<PreparePollResult> => {
    for (let attempt = 0; attempt < PREPARE_POLL_MAX; attempt += 1) {
      if (connectGenRef.current !== gen) {
        return { status: 'cancelled' };
      }
      const req = await api.getPrepareRequest(requestId);
      updateConnectFlow({
        status: req.status,
        detail: req.progress_message,
        errorCode: req.error_code,
      });
      if (isPrepareTerminal(req.status)) {
        return req;
      }
      await new Promise((resolve) => window.setTimeout(resolve, LAUNCH_POLL_MS));
    }
    return { status: 'timeout' };
  };

  const ensureOfflineProfile = async (): Promise<string | undefined> => {
    if (mojangStatus?.linked) return undefined;
    if (profiles[0]?.id) return profiles[0].id;
    const profile = await api.createProfile({
      username: offlineUsernameFromIdentity(user),
    });
    setProfiles([profile]);
    return profile.id;
  };

  const ensureBindingForServer = async (
    server: MonitoringServer,
    gen: number,
  ): Promise<MonitoringInstanceBinding | null> => {
    updateConnectFlow({ step: 'creating' });
    const binding = await api.ensureMonitoringConnectInstance(server.id);
    if (connectGenRef.current !== gen) return null;
    setBindings((prev) => new Map(prev).set(server.id, binding));
    if (binding.prepare_request_id) {
      updateConnectFlow({ step: 'preparing', status: 'queued' });
      const prepare = await pollPrepareRequest(binding.prepare_request_id, gen);
      if (connectGenRef.current !== gen) return null;
      if (prepare.status !== 'completed') {
        updateConnectFlow({
          failed: true,
          status: prepare.status === 'timeout' ? 'expired' : prepare.status,
          detail: prepare.progress_message,
          errorCode:
            prepare.error_code ||
            (prepare.status === 'timeout' ? 'PREPARE_TIMEOUT' : 'PREPARE_FAILED'),
        });
        throw new Error('prepare-failed');
      }
    }
    return binding;
  };

  const launchToServer = async (
    server: MonitoringServer,
    binding: MonitoringInstanceBinding,
    gen: number,
  ) => {
    updateConnectFlow({ step: 'launching', status: 'queued', detail: undefined });
    try {
      const useLicensed = mojangStatus?.linked === true;
      const offlineProfileId = useLicensed ? undefined : await ensureOfflineProfile();
      if (connectGenRef.current !== gen) return;
      const req = await api.createLaunchRequest({
        instance_id: binding.instance_id,
        offline_profile_id: offlineProfileId,
        use_mojang_account: useLicensed,
        join_server_address: server.address,
        join_server_port: server.port,
      });
      const result = await pollLaunchRequest(req.id, gen);
      if (connectGenRef.current !== gen) return;
      if (result.status === 'running' || result.status === 'completed') {
        message.success(t('monitoring.launchCompleted'));
        closeConnectFlow();
        return;
      }
      if (result.status === 'failed' || result.status === 'expired' || result.status === 'timeout') {
        updateConnectFlow({
          failed: true,
          status: result.status === 'timeout' ? 'expired' : result.status,
          errorCode:
            'error_code' in result && result.error_code
              ? result.error_code
              : result.status === 'timeout'
                ? 'LAUNCH_TIMEOUT'
                : 'PREPARE_FAILED',
        });
        return;
      }
    } catch {
      openMinecraftLink(server);
      if (connectGenRef.current !== gen) return;
      updateConnectFlow({ failed: true, errorCode: 'PREPARE_FAILED' });
    }
  };

  const handleConnect = async (server: MonitoringServer) => {
    if (!isAuthenticated) {
      openAuthModal('login');
      return;
    }
    if (!linkedDevice) {
      message.info(t('monitoring.connectNeedsLauncher'));
      return;
    }

    const gen = connectGenRef.current + 1;
    connectGenRef.current = gen;
    setConnectingServerId(server.id);
    setConnectFlow({ server, step: 'creating' });
    try {
      await ensureBindingForServer(server, gen);
      if (connectGenRef.current !== gen) return;
      await ensureOfflineProfile();
      if (connectGenRef.current !== gen) return;
      updateConnectFlow({
        step: 'clientMods',
        status: undefined,
        detail: undefined,
        errorCode: undefined,
      });
    } catch (e) {
      if (connectGenRef.current !== gen) return;
      if (e instanceof Error && e.message === 'prepare-failed') {
        return;
      }
      updateConnectFlow({ failed: true });
      message.error(e instanceof Error ? e.message : t('monitoring.connectInstanceFailed'));
    }
  };

  const handleConnectModsConfirmed = async () => {
    if (!connectFlow) return;
    const binding = bindings.get(connectFlow.server.id);
    if (!binding) return;
    const server = connectFlow.server;
    const gen = connectGenRef.current;
    updateConnectFlow({ step: 'syncing', status: undefined, detail: undefined });
    try {
      const prepared = await api.prepareConnectMods(server.id, binding.instance_id);
      if (connectGenRef.current !== gen) return;
      const installedCount =
        (prepared.client_mods_installed?.length ?? 0) +
        (prepared.server_mods_installed?.length ?? 0) +
        (prepared.client_resourcepacks_installed?.length ?? 0) +
        (prepared.server_resourcepacks_installed?.length ?? 0) +
        (prepared.client_shaders_installed?.length ?? 0) +
        (prepared.server_shaders_installed?.length ?? 0);
      const prepareErrors = prepared.errors ?? [];
      if (prepareErrors.length > 0) {
        const names = prepareErrors
          .map((item) => item.split(':')[0]?.trim())
          .filter(Boolean)
          .slice(0, 4)
          .join(', ');
        message.warning(
          names
            ? t('monitoring.connectMods.syncPartialDetail', { names })
            : t('monitoring.connectMods.syncPartial'),
        );
      } else if (installedCount > 0) {
        message.success(t('monitoring.connectMods.synced', { count: installedCount }));
      } else if (!prepared.agent_online && (prepared.client_mods_installed?.length ?? 0) === 0) {
        message.info(t('monitoring.connectMods.agentOffline'));
      }
      await launchToServer(server, binding, gen);
    } catch {
      if (connectGenRef.current !== gen) return;
      updateConnectFlow({ failed: true });
      message.error(t('monitoring.connectMods.syncFailed'));
      openMinecraftLink(server);
    }
  };

  const applySearch = () => {
    setFilters((prev) => ({ ...prev, q: draftQuery.trim() }));
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await loadServers();
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <div className="monitoring-page">
      <section className="monitoring-hero">
        <div className="monitoring-hero-inner">
          <div className="monitoring-hero-content">
            <Title level={1} className="monitoring-hero-title">
              {highlightMinecraft(t('monitoring.title'))}
            </Title>
            <Paragraph type="secondary" className="monitoring-hero-subtitle">{t('monitoring.subtitle')}</Paragraph>
            <div className="monitoring-hero-stats">
              <span className="monitoring-stat-pill">
                {t('monitoring.statTotal', { count: servers.length })}
              </span>
              <span className="monitoring-stat-pill monitoring-stat-pill--online">
                {t('monitoring.statOnline', { count: onlineCount })}
              </span>
              <Button
                icon={<ReloadOutlined spin={refreshing} />}
                loading={refreshing}
                onClick={() => void handleRefresh()}
              >
                {t('monitoring.refresh')}
              </Button>
            </div>
          </div>
        </div>
      </section>

      <div className="monitoring-body">
      <section className="monitoring-filters">
        <Row gutter={[12, 12]}>
          <Col xs={24} sm={12} md={6}>
            <Select
              allowClear
              showSearch
              placeholder={t('monitoring.filterVersion')}
              className="monitoring-filter-control"
              value={filters.mc_version || undefined}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, mc_version: value ?? '' }))
              }
              options={mcVersions.map((version) => ({ value: version, label: version }))}
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              allowClear
              showSearch
              placeholder={t('monitoring.filterLoader')}
              className="monitoring-filter-control"
              value={filters.loader || undefined}
              onChange={(value) => setFilters((prev) => ({ ...prev, loader: value ?? '' }))}
              options={ALL_GAME_SERVER_TYPES.map((type) => ({
                value: type,
                label: loaderLabel(type),
              }))}
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              allowClear
              showSearch
              placeholder={t('monitoring.filterMod')}
              className="monitoring-filter-control"
              value={filters.mod || undefined}
              onChange={(value) => setFilters((prev) => ({ ...prev, mod: value ?? '' }))}
              options={modOptions.map((mod) => ({ value: mod, label: mod }))}
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              allowClear
              showSearch
              placeholder={t('monitoring.filterPlugin')}
              className="monitoring-filter-control"
              value={filters.plugin || undefined}
              onChange={(value) => setFilters((prev) => ({ ...prev, plugin: value ?? '' }))}
              options={pluginOptions.map((plugin) => ({ value: plugin, label: plugin }))}
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              placeholder={t('monitoring.sortLabel')}
              className="monitoring-filter-control"
              value={sortBy}
              onChange={setSortBy}
              options={sortOptions}
            />
          </Col>
          <Col xs={24} md={16}>
            <Input
              allowClear
              prefix={<SearchOutlined aria-hidden />}
              placeholder={t('monitoring.searchPlaceholder')}
              value={draftQuery}
              onChange={(e) => setDraftQuery(e.target.value)}
              onPressEnter={applySearch}
              className="monitoring-filter-control"
            />
          </Col>
          <Col xs={24} md={8}>
            <Space wrap>
              <Button type="primary" onClick={applySearch}>
                {t('monitoring.search')}
              </Button>
              <Button
                onClick={() => {
                  setDraftQuery('');
                  setSortBy('online');
                  setFilters(EMPTY_FILTERS);
                }}
              >
                {t('monitoring.resetFilters')}
              </Button>
            </Space>
          </Col>
        </Row>
      </section>

      <section className="monitoring-list">
        {loading ? (
          <div className="monitoring-loading">
            <Spin size="large" />
          </div>
        ) : sortedServers.length === 0 ? (
          <Empty description={t('monitoring.empty')} className="monitoring-empty" />
        ) : (
          <div className="monitoring-cards">
            {sortedServers.map((server) => (
              <MonitoringServerCard
                key={server.id}
                server={server}
                liked={likedIds.has(server.id)}
                onLike={(item) => void handleLike(item)}
                onRate={(item, rating) => void handleRate(item, rating)}
                canInteract={isAuthenticated}
                isGuest={!isAuthenticated}
                onRequireAuth={() => openAuthModal('login')}
                loaderLabel={loaderLabel(server.server_type)}
                boundInstanceName={bindings.get(server.id)?.instance_name}
                onConnect={(item) => void handleConnect(item)}
                connecting={connectingServerId === server.id}
              />
            ))}
          </div>
        )}
      </section>
      </div>
      {connectFlow ? (
        <ConnectProgressModal
          open
          serverName={connectFlow.server.name}
          step={connectFlow.step}
          status={connectFlow.status}
          detail={connectFlow.detail}
          errorCode={connectFlow.errorCode}
          failed={connectFlow.failed}
          onClose={closeConnectFlow}
        >
          {connectFlow.step === 'clientMods' && bindings.get(connectFlow.server.id) ? (
            <ConnectClientModsModal
              embedded
              open
              gameServerId={connectFlow.server.id}
              instanceId={bindings.get(connectFlow.server.id)!.instance_id}
              serverName={connectFlow.server.name}
              onClose={closeConnectFlow}
              onConfirm={() => void handleConnectModsConfirmed()}
            />
          ) : null}
        </ConnectProgressModal>
      ) : null}
    </div>
  );
}
