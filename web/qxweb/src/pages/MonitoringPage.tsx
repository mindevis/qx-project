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
  HeartFilled,
  HeartOutlined,
  PlayCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { api, type MonitoringServer } from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { ALL_GAME_SERVER_TYPES, gameServerTypeLabelText } from '@/lib/gameServerTypes';
import { highlightMinecraft } from '@/pages/HomePage';
import './MonitoringPage.css';

const { Title, Paragraph, Text } = Typography;

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
}: {
  server: MonitoringServer;
  liked: boolean;
  onLike: (server: MonitoringServer) => void;
  onRate: (server: MonitoringServer, rating: number) => void;
  canInteract: boolean;
  onRequireAuth: () => void;
  loaderLabel: string;
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
        <span
          className={[
            'monitoring-card-status',
            server.is_online ? 'monitoring-card-status--online' : 'monitoring-card-status--offline',
          ].join(' ')}
        >
          {server.is_online ? t('monitoring.online') : t('monitoring.offline')}
        </span>
      </div>

      <div className="monitoring-card-body">
        <div className="monitoring-card-head">
          <Title level={4} className="monitoring-card-title">
            {server.name}
          </Title>
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
              href={`minecraft://${server.address}:${server.port}`}
            >
              {t('monitoring.connect')}
            </Button>
          </Space>
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
  const [servers, setServers] = useState<MonitoringServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [likedIds, setLikedIds] = useState<Set<string>>(new Set());
  const [mcVersions, setMcVersions] = useState<string[]>([]);

  const loaderLabel = useCallback(
    (type: string) => gameServerTypeLabelText(t, type),
    [t],
  );

  useEffect(() => {
    void api
      .listMcVersions()
      .then((data) => {
        const versions = [...new Set((data.items ?? []).map((item) => item.id))].sort().reverse();
        setMcVersions(versions);
      })
      .catch(() => {
        /* optional filter source */
      });
  }, []);

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

  const applySearch = () => {
    setFilters((prev) => ({ ...prev, q: draftQuery.trim() }));
  };

  return (
    <div className="monitoring-page">
      <section className="monitoring-hero">
        <p className="monitoring-hero-badge">{t('monitoring.badge')}</p>
        <Title level={1} className="monitoring-hero-title">
          {highlightMinecraft(t('monitoring.title'))}
        </Title>
        <Paragraph className="monitoring-hero-subtitle">{t('monitoring.subtitle')}</Paragraph>
      </section>

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
        ) : servers.length === 0 ? (
          <Empty description={t('monitoring.empty')} className="monitoring-empty" />
        ) : (
          <div className="monitoring-cards">
            {servers.map((server) => (
              <MonitoringServerCard
                key={server.id}
                server={server}
                liked={likedIds.has(server.id)}
                onLike={(item) => void handleLike(item)}
                onRate={(item, rating) => void handleRate(item, rating)}
                canInteract={isAuthenticated}
                onRequireAuth={() => openAuthModal('login')}
                loaderLabel={loaderLabel(server.server_type)}
              />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
