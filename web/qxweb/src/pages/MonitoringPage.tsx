import { useCallback, useEffect, useMemo, useState } from 'react';
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
  GlobalOutlined,
  HeartFilled,
  HeartOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import {
  api,
  type LauncherInstance,
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
import { isLaunchTerminal } from '@/lib/launchProgress';
import { cachedListMcVersions } from '@/lib/mcVersionsCache';
import { ConnectClientModsModal } from '@/components/ConnectClientModsModal';
import { highlightMinecraft } from '@/pages/HomePage';
import './MonitoringPage.css';

const { Title, Paragraph, Text } = Typography;

const LAUNCH_POLL_MS = 1500;
const LAUNCH_POLL_MAX = 120;

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
  instances,
  boundInstanceId,
  onBindingChange,
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
  instances: LauncherInstance[];
  boundInstanceId?: string;
  onBindingChange: (server: MonitoringServer, instanceId: string | null) => void;
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

  const handleBindingSelect = (value: string | null) => {
    if (!canInteract) {
      onRequireAuth();
      return;
    }
    onBindingChange(server, value);
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

        {canInteract && instances.length > 0 ? (
          <div className="monitoring-card-binding">
            <Text type="secondary" className="monitoring-card-binding-label">
              {t('monitoring.bindInstance')}
            </Text>
            <Select
              allowClear
              showSearch
              aria-label={t('monitoring.bindInstance')}
              placeholder={t('monitoring.bindInstancePlaceholder')}
              className="monitoring-card-binding-select"
              value={boundInstanceId}
              onChange={handleBindingSelect}
              optionFilterProp="label"
              options={instances.map((instance) => ({
                value: instance.id,
                label: `${instance.name} (${instance.mc_version})`,
              }))}
            />
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
  const { isAuthenticated } = useAuth();
  const { openAuthModal } = useAuthModal();
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [draftQuery, setDraftQuery] = useState('');
  const [sortBy, setSortBy] = useState<SortBy>('online');
  const [servers, setServers] = useState<MonitoringServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [likedIds, setLikedIds] = useState<Set<string>>(new Set());
  const [mcVersions, setMcVersions] = useState<string[]>([]);
  const [instances, setInstances] = useState<LauncherInstance[]>([]);
  const [bindings, setBindings] = useState<Map<string, MonitoringInstanceBinding>>(new Map());
  const [linkedDevice, setLinkedDevice] = useState<{ device_id: string } | null>(null);
  const [profiles, setProfiles] = useState<OfflineProfile[]>([]);
  const [mojangStatus, setMojangStatus] = useState<MojangLinkStatus | null>(null);
  const [connectingServerId, setConnectingServerId] = useState<string | null>(null);
  const [connectModsServer, setConnectModsServer] = useState<MonitoringServer | null>(null);
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
      setInstances([]);
      setBindings(new Map());
      setLinkedDevice(null);
      setProfiles([]);
      setMojangStatus(null);
      return;
    }
    void (async () => {
      try {
        const [instanceData, bindingData, device, profileData, mojang] = await Promise.all([
          api.listInstances(),
          api.listMonitoringBindings(),
          api.myLauncherDevice().catch(() => null),
          api.listProfiles().catch(() => ({ items: [] as OfflineProfile[] })),
          api.mojangStatus().catch(() => null),
        ]);
        setInstances(instanceData.items ?? []);
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

  const handleBindingChange = async (server: MonitoringServer, instanceId: string | null) => {
    try {
      if (!instanceId) {
        await api.clearMonitoringBinding(server.id);
        setBindings((prev) => {
          const next = new Map(prev);
          next.delete(server.id);
          return next;
        });
        message.success(t('monitoring.bindingCleared'));
        return;
      }
      const binding = await api.setMonitoringBinding(server.id, instanceId);
      setBindings((prev) => new Map(prev).set(server.id, binding));
      message.success(t('monitoring.bindingSaved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    }
  };

  const openMinecraftLink = (server: MonitoringServer) => {
    window.location.href = `minecraft://${server.address}:${server.port}`;
  };

  const pollLaunchRequest = async (requestId: string) => {
    for (let attempt = 0; attempt < LAUNCH_POLL_MAX; attempt += 1) {
      await new Promise((resolve) => window.setTimeout(resolve, LAUNCH_POLL_MS));
      const req = await api.getLaunchRequest(requestId);
      if (isLaunchTerminal(req.status)) {
        return;
      }
    }
  };

  const launchToServer = async (server: MonitoringServer, binding: MonitoringInstanceBinding) => {
    setConnectingServerId(server.id);
    try {
      const useLicensed = mojangStatus?.linked === true;
      const req = await api.createLaunchRequest({
        instance_id: binding.instance_id,
        offline_profile_id: useLicensed ? undefined : profiles[0]?.id,
        use_mojang_account: useLicensed,
        join_server_address: server.address,
        join_server_port: server.port,
      });
      await pollLaunchRequest(req.id);
    } catch {
      openMinecraftLink(server);
    } finally {
      setConnectingServerId(null);
    }
  };

  const handleConnect = async (server: MonitoringServer) => {
    const binding = bindings.get(server.id);
    if (!isAuthenticated || !binding || !linkedDevice) {
      openMinecraftLink(server);
      if (isAuthenticated && binding && !linkedDevice) {
        message.info(t('monitoring.connectNeedsLauncher'));
      }
      return;
    }

    setConnectModsServer(server);
  };

  const handleConnectModsConfirmed = async () => {
    if (!connectModsServer) return;
    const binding = bindings.get(connectModsServer.id);
    if (!binding) return;
    const server = connectModsServer;
    setConnectModsServer(null);
    setConnectingServerId(server.id);
    try {
      const prepared = await api.prepareConnectMods(server.id, binding.instance_id);
      const installedCount =
        (prepared.client_mods_installed?.length ?? 0) +
        (prepared.server_mods_installed?.length ?? 0) +
        (prepared.client_resourcepacks_installed?.length ?? 0) +
        (prepared.server_resourcepacks_installed?.length ?? 0) +
        (prepared.client_shaders_installed?.length ?? 0) +
        (prepared.server_shaders_installed?.length ?? 0);
      if (installedCount > 0) {
        message.success(t('monitoring.connectMods.synced', { count: installedCount }));
      } else if ((prepared.errors?.length ?? 0) > 0) {
        message.warning(t('monitoring.connectMods.syncPartial'));
      } else if (!prepared.agent_online && (prepared.client_mods_installed?.length ?? 0) === 0) {
        message.info(t('monitoring.connectMods.agentOffline'));
      }
      await launchToServer(server, binding);
    } catch {
      message.error(t('monitoring.connectMods.syncFailed'));
      openMinecraftLink(server);
    } finally {
      setConnectingServerId(null);
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
        <div className="monitoring-hero-ambient" aria-hidden>
          <span className="monitoring-hero-blob monitoring-hero-blob--1" />
          <span className="monitoring-hero-blob monitoring-hero-blob--2" />
          <span className="monitoring-hero-blob monitoring-hero-blob--3" />
          <span className="monitoring-hero-grid-pattern" />
        </div>

        <div className="monitoring-hero-inner">
          <div className="monitoring-hero-content">
            <p className="monitoring-hero-badge">{t('monitoring.badge')}</p>
            <Title level={1} className="monitoring-hero-title">
              {highlightMinecraft(t('monitoring.title'))}
            </Title>
            <Paragraph className="monitoring-hero-subtitle">{t('monitoring.subtitle')}</Paragraph>
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

          <div className="monitoring-hero-visual" aria-hidden>
            <div className="monitoring-orbit">
              <div className="monitoring-orbit-ring monitoring-orbit-ring--outer" />
              <div className="monitoring-orbit-ring monitoring-orbit-ring--inner" />
              <div className="monitoring-orbit-core">
                <GlobalOutlined />
              </div>
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
                instances={instances}
                boundInstanceId={bindings.get(server.id)?.instance_id}
                onBindingChange={(item, instanceId) => void handleBindingChange(item, instanceId)}
                onConnect={(item) => void handleConnect(item)}
                connecting={connectingServerId === server.id}
              />
            ))}
          </div>
        )}
      </section>
      </div>
      {connectModsServer && bindings.get(connectModsServer.id) ? (
        <ConnectClientModsModal
          open
          gameServerId={connectModsServer.id}
          instanceId={bindings.get(connectModsServer.id)!.instance_id}
          serverName={connectModsServer.name}
          onClose={() => setConnectModsServer(null)}
          onConfirm={handleConnectModsConfirmed}
        />
      ) : null}
    </div>
  );
}
