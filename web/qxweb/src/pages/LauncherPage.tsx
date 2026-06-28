import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import {
  CheckCircleOutlined,
  DeleteOutlined,
  DesktopOutlined,
  DownloadOutlined,
  LinkOutlined,
  LoginOutlined,
  PlusOutlined,
  RocketOutlined,
  ReloadOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { LauncherDownloadButton } from '@/components/LauncherDownloadButton';
import { ProfileModelPicker, ProfileModelAvatar } from '@/components/ProfileModelPicker';
import { useMessage } from '@/hooks/useMessage';
import {
  DEFAULT_MC_VERSION,
  fallbackMcVersionsList,
  groupMcVersionOptions,
  pickDefaultMcVersion,
  type McVersionItem,
} from '@/launcher/mcVersions';
import {
  DEFAULT_LAUNCHER_LOADER,
  LAUNCHER_LOADERS,
  launcherLoaderAsGameServerType,
  launcherLoaderNeedsVersion,
  isLauncherLoader,
  type LauncherLoader,
} from '@/lib/launcherLoaders';
import {
  listGameServerLoaderVersions,
  listGameServerMcVersions,
  mcVersionOptionsFromItems,
  type VersionOption,
} from '@/lib/gameServerVersions';
import {
  api,
  clearLinkedDevice,
  saveLinkedDevice,
  type LauncherInstance,
  type OfflineProfile,
} from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { getLaunchStatusKey } from '@/i18n';
import { useI18n } from '@/i18n/I18nContext';
import { modalMotionProps } from '@/lib/modal';
import { logger } from '@/lib/logger';
import './LauncherPage.css';

const { Title, Paragraph, Text } = Typography;

const LAUNCH_POLL_MS = 1500;
const LAUNCH_TERMINAL = new Set(['completed', 'failed', 'expired']);

export function LauncherPage() {
  const { t } = useI18n();
  const { isAuthenticated, user, loading: authLoading } = useAuth();
  const { openAuthModal } = useAuthModal();
  const message = useMessage();
  const [instances, setInstances] = useState<LauncherInstance[]>([]);
  const [profiles, setProfiles] = useState<OfflineProfile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [profilesLoading, setProfilesLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<{
    name: string;
    loader: LauncherLoader;
    mc_version: string;
    loader_version?: string;
  }>();
  const [profileOpen, setProfileOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [creatingProfile, setCreatingProfile] = useState(false);
  const [mcVersions, setMcVersions] = useState<McVersionItem[]>([]);
  const [defaultMcVersion, setDefaultMcVersion] = useState(DEFAULT_MC_VERSION);
  const [mcVersionsLoading, setMcVersionsLoading] = useState(false);
  const [createLoader, setCreateLoader] = useState<LauncherLoader>(DEFAULT_LAUNCHER_LOADER);
  const [createMcOptions, setCreateMcOptions] = useState<VersionOption[]>([]);
  const [createLoaderOptions, setCreateLoaderOptions] = useState<VersionOption[]>([]);
  const [createMcOptionsLoading, setCreateMcOptionsLoading] = useState(false);
  const [createLoaderOptionsLoading, setCreateLoaderOptionsLoading] = useState(false);
  const [launchingId, setLaunchingId] = useState<string | null>(null);
  const [linkedDevice, setLinkedDevice] = useState<{ device_id: string; status: string } | null>(
    null,
  );
  const [refreshing, setRefreshing] = useState(false);
  const [, setAccessKey] = useState(0);
  const refreshAccess = useCallback(() => setAccessKey((k) => k + 1), []);
  const canManage = !authLoading && isAuthenticated;
  const instancesTitle = t('launcher.myInstances');
  const selectedProfile = profiles.find((p) => p.id === selectedProfileId);
  const activePlayerLabel = selectedProfile?.username ?? t('launcher.playerDefault');

  const formatDeviceId = (deviceId: string) =>
    deviceId.length > 16 ? `${deviceId.slice(0, 8)}…${deviceId.slice(-4)}` : deviceId;

  const mcVersionTypeLabel = useCallback(
    (type: string) => {
      const key = `launcher.mcVersionType.${type}`;
      const label = t(key);
      return label === key ? type : label;
    },
    [t],
  );

  const createNeedsLoader = launcherLoaderNeedsVersion(createLoader);

  const loaderLabel = useCallback(
    (loader: LauncherLoader) => {
      const key = `servers.gameServerType.${loader}`;
      const label = t(key);
      return label === key ? loader : label;
    },
    [t],
  );

  const loaderOptions = LAUNCHER_LOADERS.map((loader) => ({
    value: loader,
    label: loaderLabel(loader),
  }));

  const loadCreateMcVersions = useCallback(
    async (loader: LauncherLoader) => {
      setCreateMcOptionsLoading(true);
      try {
        const options = await listGameServerMcVersions(launcherLoaderAsGameServerType(loader), mcVersions);
        setCreateMcOptions(options);
        const current = createForm.getFieldValue('mc_version');
        const nextMc =
          options.find((o) => o.value === current)?.value ??
          options.find((o) => o.value === defaultMcVersion)?.value ??
          options[0]?.value;
        if (nextMc) {
          createForm.setFieldValue('mc_version', nextMc);
        }
        return nextMc;
      } catch (e) {
        logger.warn('failed to load create mc versions', { error: String(e), loader });
        const fallback = mcVersionOptionsFromItems(mcVersions);
        setCreateMcOptions(fallback);
        return createForm.getFieldValue('mc_version') as string | undefined;
      } finally {
        setCreateMcOptionsLoading(false);
      }
    },
    [createForm, defaultMcVersion, mcVersions],
  );

  const loadCreateLoaderVersions = useCallback(
    async (loader: LauncherLoader, mcVersion: string) => {
      if (!launcherLoaderNeedsVersion(loader) || !mcVersion) {
        setCreateLoaderOptions([]);
        createForm.setFieldValue('loader_version', undefined);
        return;
      }
      setCreateLoaderOptionsLoading(true);
      try {
        const options = await listGameServerLoaderVersions(
          launcherLoaderAsGameServerType(loader),
          mcVersion,
        );
        setCreateLoaderOptions(options);
        const current = createForm.getFieldValue('loader_version') as string | undefined;
        const nextLoader =
          options.find((o) => o.value === current)?.value ?? options[0]?.value ?? undefined;
        createForm.setFieldValue('loader_version', nextLoader ?? null);
      } catch (e) {
        logger.warn('failed to load create loader versions', { error: String(e), loader, mcVersion });
        setCreateLoaderOptions([]);
        createForm.setFieldValue('loader_version', undefined);
      } finally {
        setCreateLoaderOptionsLoading(false);
      }
    },
    [createForm],
  );

  const handleCreateLoaderChange = useCallback(
    async (loader: LauncherLoader) => {
      setCreateLoader(loader);
      createForm.setFieldValue('loader', loader);
      createForm.setFieldValue('loader_version', undefined);
      const mcVersion = await loadCreateMcVersions(loader);
      if (mcVersion) {
        await loadCreateLoaderVersions(loader, mcVersion);
      }
    },
    [createForm, loadCreateLoaderVersions, loadCreateMcVersions],
  );

  const handleCreateMcVersionChange = useCallback(
    async (mcVersion: string) => {
      await loadCreateLoaderVersions(createLoader, mcVersion);
    },
    [createLoader, loadCreateLoaderVersions],
  );

  const mcVersionOptions = groupMcVersionOptions(mcVersions, mcVersionTypeLabel);

  const loadMcVersions = useCallback(async () => {
    setMcVersionsLoading(true);
    try {
      const result = await api.listMcVersions();
      const items = result.items ?? [];
      setMcVersions(items);
      setDefaultMcVersion(pickDefaultMcVersion(result.latest, items));
    } catch (e) {
      logger.warn('failed to load mc versions', { error: String(e) });
      message.warning(t('launcher.mcVersionsLoadFailed'));
      const fallback = fallbackMcVersionsList();
      setMcVersions(fallback.items);
      setDefaultMcVersion(pickDefaultMcVersion(undefined, fallback.items));
    } finally {
      setMcVersionsLoading(false);
    }
  }, [message, t]);

  const openCreateModal = useCallback(() => {
    setCreateOpen(true);
    setCreateLoader(DEFAULT_LAUNCHER_LOADER);
    createForm.setFieldsValue({
      loader: DEFAULT_LAUNCHER_LOADER,
      mc_version: defaultMcVersion,
      loader_version: undefined,
    });
    void (async () => {
      const mcVersion = await loadCreateMcVersions(DEFAULT_LAUNCHER_LOADER);
      if (mcVersion) {
        await loadCreateLoaderVersions(DEFAULT_LAUNCHER_LOADER, mcVersion);
      }
    })();
  }, [createForm, defaultMcVersion, loadCreateLoaderVersions, loadCreateMcVersions]);

  const openProfileModal = useCallback(() => {
    setProfileOpen(true);
  }, []);

  const linkSteps = [
    {
      icon: <DesktopOutlined />,
      body: t('launcher.linkStep1'),
    },
    { icon: <LinkOutlined />, body: t('launcher.linkStep2') },
    { icon: <LoginOutlined />, body: t('launcher.linkStep3') },
    { icon: <CheckCircleOutlined />, body: t('launcher.linkStep4') },
  ];

  const launchStatusMessage = (status: string) => {
    const key = getLaunchStatusKey(status);
    const msg = t(key);
    return msg === key ? status : msg;
  };

  const loadInstances = useCallback(async () => {
    if (!canManage) {
      setInstances([]);
      return;
    }
    setLoading(true);
    try {
      const res = await api.listInstances();
      setInstances(res.items ?? []);
    } catch (e) {
      logger.warn('failed to load instances', { error: String(e) });
      message.error(t('launcher.loadInstancesFailed'));
    } finally {
      setLoading(false);
    }
  }, [canManage, message, t]);

  const loadProfiles = useCallback(async () => {
    if (!canManage) {
      setProfiles([]);
      setSelectedProfileId(undefined);
      return;
    }
    setProfilesLoading(true);
    try {
      const res = await api.listProfiles();
      const items = res.items ?? [];
      setProfiles(items);
      setSelectedProfileId((prev) => {
        /* v8 ignore next 3 -- @preserve */
        if (prev && items.some((p) => p.id === prev)) return prev;
        return items[0]?.id;
      });
    } catch (e) {
      logger.warn('failed to load profiles', { error: String(e) });
      message.error(t('launcher.loadProfilesFailed'));
    } finally {
      setProfilesLoading(false);
    }
  }, [canManage, message, t]);

  /* v8 ignore start -- @preserve workspace refresh is covered via instance/profile reload paths */
  const refreshWorkspace = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([loadInstances(), loadProfiles()]);
      message.success(t('launcher.workspaceRefreshed'));
    } finally {
      setRefreshing(false);
    }
  }, [loadInstances, loadProfiles, message, t]);
  /* v8 ignore end */

  useEffect(() => {
    void loadInstances();
    void loadProfiles();
  }, [loadInstances, loadProfiles]);

  useEffect(() => {
    if (canManage) {
      void loadMcVersions();
    }
  }, [canManage, loadMcVersions]);

  useEffect(() => {
    const onAccessChange = () => refreshAccess();
    window.addEventListener('storage', onAccessChange);
    window.addEventListener('focus', onAccessChange);
    return () => {
      window.removeEventListener('storage', onAccessChange);
      window.removeEventListener('focus', onAccessChange);
    };
  }, [refreshAccess]);

  useEffect(() => {
    if (!isAuthenticated) {
      setLinkedDevice(null);
      return;
    }
    (async () => {
      try {
        const res = await api.myLauncherDevice();
        if (res.linked && res.device_id) {
          setLinkedDevice({ device_id: res.device_id, status: res.status ?? 'linked' });
          saveLinkedDevice(res.device_id);
        } else {
          setLinkedDevice(null);
        }
      } catch (e) {
        logger.warn('failed to load linked device', { error: String(e) });
      }
    })();
  }, [isAuthenticated]);

  const handleCreate = async (values: {
    name: string;
    loader: LauncherLoader;
    mc_version: string;
    loader_version?: string;
  }) => {
    setCreating(true);
    try {
      await api.createInstance({
        name: values.name,
        mc_version: values.mc_version,
        loader: values.loader,
        loader_version: launcherLoaderNeedsVersion(values.loader)
          ? values.loader_version?.trim() || undefined
          : undefined,
      });
      message.success(t('launcher.instanceCreated'));
      setCreateOpen(false);
      await loadInstances();
      const prof = await api.listProfiles();
      if ((prof.items ?? []).length === 0) {
        message.info(t('launcher.createProfileHint'));
        setProfileOpen(true);
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('launcher.createInstanceFailed'));
    } finally {
      setCreating(false);
    }
  };

  const handleCreateProfile = async (values: { username: string; model: 'steve' | 'alex' }) => {
    setCreatingProfile(true);
    try {
      const profile = await api.createProfile({
        username: values.username,
        model: values.model ?? 'steve',
      });
      message.success(t('launcher.profileCreated'));
      setProfileOpen(false);
      setProfiles((prev) => [...prev, profile]);
      setSelectedProfileId(profile.id);
    } catch (e) {
      if (e instanceof Error) {
        message.error(e.message);
      } else {
        message.error(t('launcher.createProfileFailed'));
      }
    } finally {
      setCreatingProfile(false);
    }
  };

  const handleDeleteProfile = async (id: string) => {
    try {
      await api.deleteProfile(id);
      message.success(t('launcher.profileDeleted'));
      setProfiles((prev) => prev.filter((p) => p.id !== id));
      setSelectedProfileId((prev) => (prev === id ? undefined : /* v8 ignore next -- @preserve */ prev));
    } catch (e) {
      if (e instanceof Error) {
        message.error(e.message);
      } else {
        message.error(t('launcher.deleteProfileFailed'));
      }
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await api.deleteInstance(id);
      message.success(t('launcher.instanceDeleted'));
      await loadInstances();
    } catch (e) {
      if (e instanceof Error) {
        message.error(e.message);
      } else {
        message.error(t('launcher.deleteFailed'));
      }
    }
  };

  const pollLaunchRequest = async (requestId: string) => {
    const started = Date.now();
    const timeoutMs = 30 * 60 * 1000;
    let lastStatus: string | undefined;
    while (Date.now() - started < timeoutMs) {
      const req = await api.getLaunchRequest(requestId);
      if (req.status !== lastStatus) {
        lastStatus = req.status;
        message.info(launchStatusMessage(req.status), 2);
      }
      if (LAUNCH_TERMINAL.has(req.status)) {
        if (req.status === 'completed') {
          message.success(t('launcher.gameLaunched'));
        } else if (req.status === 'failed') {
          const errorKey =
            req.error_code === 'LOADER_INSTALL_FAILED'
              ? 'launcher.launchErrorLoaderInstall'
              : req.error_code === 'JAVA_FAILED' || req.error_code === 'JAVA_START_FAILED'
                ? 'launcher.launchErrorJava'
                : undefined;
          message.error(errorKey ? t(errorKey) : (req.error_code ?? t('launcher.launchError')));
        } else {
          message.warning(launchStatusMessage(req.status));
        }
        return;
      }
      /* v8 ignore next 3 -- @preserve */
      await new Promise((r) => setTimeout(r, LAUNCH_POLL_MS));
    }
    /* v8 ignore next -- @preserve */
    message.warning(t('launcher.launchTimeout'));
  };

  const handleUnlinkDevice = async () => {
    try {
      await api.unlinkDevice();
      setLinkedDevice(null);
      clearLinkedDevice();
      message.success(t('launcher.launcherUnlinked'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('launcher.unlinkFailed'));
    }
  };

  const handlePlay = async (instance: LauncherInstance) => {
    setLaunchingId(instance.id);
    try {
      if (!selectedProfileId) {
        message.info(t('launcher.defaultPlayerHint'));
      }
      const req = await api.createLaunchRequest({
        instance_id: instance.id,
        offline_profile_id: selectedProfileId,
      });
      message.info(t('launcher.launchSent'));
      await pollLaunchRequest(req.id);
    } catch (e) {
      if (e instanceof Error) {
        message.error(e.message);
      } else {
        message.error(t('launcher.launchGameFailed'));
      }
    } finally {
      setLaunchingId(null);
    }
  };

  return (
    <div className="launcher-page">
      <section className="launcher-hero">
        <div className="launcher-hero-ambient" aria-hidden>
          <span className="launcher-hero-blob launcher-hero-blob--1" />
          <span className="launcher-hero-blob launcher-hero-blob--2" />
          <span className="launcher-hero-blob launcher-hero-blob--3" />
          <span className="launcher-hero-grid-pattern" />
        </div>

        <div className="launcher-hero-inner">
          <div className="launcher-hero-content">
            <span className="launcher-badge">{t('home.heroTagLauncher')}</span>
            <Title level={1} className="launcher-title">
              <span className="launcher-title-highlight">{t('launcher.title')}</span>
            </Title>
            <Paragraph className="launcher-intro">{t('launcher.intro')}</Paragraph>
            {canManage ? (
              <div className="launcher-hero-actions">
                <Button
                  type="primary"
                  size="large"
                  icon={<PlusOutlined />}
                  disabled={!canManage}
                  onClick={openCreateModal}
                >
                  {t('launcher.heroCreateInstance')}
                </Button>
                <Button
                  size="large"
                  icon={<UserOutlined />}
                  onClick={openProfileModal}
                >
                  {t('launcher.heroAddProfile')}
                </Button>
              </div>
            ) : null}
          </div>

          <div className="launcher-hero-visual" aria-hidden>
            <div className="launcher-orbit">
              <div className="launcher-orbit-ring launcher-orbit-ring--outer" />
              <div className="launcher-orbit-ring launcher-orbit-ring--inner" />
              <div className="launcher-orbit-core">
                <DesktopOutlined />
              </div>
            </div>
          </div>
        </div>
      </section>

      {!authLoading && !canManage && (
        <section className="launcher-section launcher-section--steps">
          <div className="launcher-section-header">
            <span className="launcher-section-eyebrow">{t('home.heroTagLauncher')}</span>
            <Title level={2} className="launcher-section-title">
              {t('launcher.linkFirstTitle')}
            </Title>
          </div>
          <div className="launcher-steps">
            {linkSteps.map((step, index) => (
              <div key={step.body} className="launcher-step">
                <span className="launcher-step-number">{index + 1}</span>
                <span className="launcher-step-icon">{step.icon}</span>
                <Paragraph className="launcher-step-body">{step.body}</Paragraph>
              </div>
            ))}
          </div>
          <div className="launcher-steps-actions">
            <Button icon={<ReloadOutlined />} onClick={refreshAccess}>
              {t('launcher.checkLink')}
            </Button>
            <Button type="link" onClick={() => openAuthModal('login')}>
              {t('launcher.signInToAccount')}
            </Button>
          </div>
        </section>
      )}

      {isAuthenticated && user && (
        <section className="launcher-section launcher-section--status">
          <div
            className={`launcher-status-card launcher-status-card--${linkedDevice ? 'linked' : 'warning'}`}
          >
            <div className="launcher-status-icon">
              {linkedDevice ? <CheckCircleOutlined /> : <LinkOutlined />}
            </div>
            <div className="launcher-status-content">
              <Text strong className="launcher-status-title">
                {t('launcher.accountLabel', { email: user.email })}
              </Text>
              <Paragraph className="launcher-status-desc">
                {linkedDevice
                  ? t('launcher.linkedDesc', { deviceId: formatDeviceId(linkedDevice.device_id) })
                  : t('launcher.notLinkedDesc')}
              </Paragraph>
            </div>
            {linkedDevice ? (
              <Popconfirm
                title={t('launcher.unlinkConfirm')}
                onConfirm={() => void handleUnlinkDevice()}
              >
                <Button danger>{t('launcher.unlink')}</Button>
              </Popconfirm>
            ) : (
              <Link to="/launcher/link">
                <Button type="primary" icon={<LinkOutlined />}>
                  {t('launcher.openLinkPage')}
                </Button>
              </Link>
            )}
          </div>
        </section>
      )}

      <section className="launcher-section launcher-section--workspace">
        <div className="launcher-section-header launcher-section-header--left">
          <div className="launcher-section-header-row">
            <div>
              <span className="launcher-section-eyebrow">{t('home.heroTagLauncher')}</span>
              <Title level={2} className="launcher-section-title">
                {t('launcher.workspaceTitle')}
              </Title>
              <Paragraph type="secondary" className="launcher-section-lead">
                {t('launcher.workspaceSubtitle')}
              </Paragraph>
            </div>
            {canManage ? (
              <Button
                icon={<ReloadOutlined spin={refreshing} />}
                loading={refreshing}
                onClick={() => void refreshWorkspace()}
              >
                {t('launcher.workspaceRefresh')}
              </Button>
            ) : null}
          </div>
          {canManage ? (
            <div className="launcher-workspace-stats">
              <span className="launcher-stat-pill">
                {t('launcher.statInstances', { count: instances.length })}
              </span>
              <span className="launcher-stat-pill">
                {t('launcher.statProfiles', { count: profiles.length })}
              </span>
            </div>
          ) : null}
        </div>

        {!linkedDevice && isAuthenticated && (
          <div className="launcher-download-band">
            <div className="launcher-download-band-icon">
              <DownloadOutlined />
            </div>
            <div className="launcher-download-band-text">
              <Title level={5} className="launcher-download-band-title">
                {t('launcher.desktopTitle')}
              </Title>
              <Paragraph className="launcher-download-band-desc">{t('launcher.desktopDesc')}</Paragraph>
            </div>
            <LauncherDownloadButton type="primary" />
          </div>
        )}

        <div className="launcher-workspace-grid">
          <div className="launcher-panel launcher-panel--instances">
            <div className="launcher-panel-header">
              <div className="launcher-panel-heading">
                <span className="launcher-panel-icon launcher-panel-icon--play">
                  <RocketOutlined />
                </span>
                <div>
                  <Title level={4} className="launcher-panel-title">
                    {instancesTitle}
                  </Title>
                  {instances.length > 0 ? (
                    <Text type="secondary" className="launcher-panel-count">
                      {instances.length}
                    </Text>
                  ) : null}
                </div>
              </div>
              {canManage && instances.length > 0 ? (
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
                  {t('common.create')}
                </Button>
              ) : null}
            </div>

            {authLoading ? (
              <div className="launcher-panel-loading">
                <Spin />
              </div>
            ) : !canManage ? (
              <Paragraph type="secondary" className="launcher-panel-empty">
                {t('launcher.signInRequired')}
              </Paragraph>
            ) : loading ? (
              <div className="launcher-panel-loading">
                <Spin />
              </div>
            ) : instances.length === 0 ? (
              <div className="launcher-empty">
                <RocketOutlined className="launcher-empty-icon" />
                <Paragraph type="secondary">{t('launcher.noInstances')}</Paragraph>
                {canManage ? (
                  <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
                    {t('launcher.createFirstInstance')}
                  </Button>
                ) : null}
              </div>
            ) : (
              <>
                <div className="launcher-launch-bar">
                  <Text type="secondary">{t('launcher.playingAs')}</Text>
                  <Text strong className="launcher-launch-bar-name">
                    {activePlayerLabel}
                  </Text>
                  {selectedProfile ? (
                    <ProfileModelAvatar model={selectedProfile.model ?? 'steve'} size="sm" />
                  ) : null}
                </div>
                <div className="launcher-instance-list">
                  {instances.map((item) => (
                    <div
                      key={item.id}
                      className={`launcher-instance-card${launchingId === item.id ? ' launcher-instance-card--launching' : ''}`}
                    >
                      <div className="launcher-instance-info">
                        <Text strong className="launcher-instance-name">
                          {item.name}
                        </Text>
                        <div className="launcher-instance-tags">
                          <span className="launcher-tag launcher-tag--version">
                            Minecraft {item.mc_version}
                          </span>
                          <span className="launcher-tag">
                            {isLauncherLoader(item.loader) ? loaderLabel(item.loader) : item.loader}
                          </span>
                          {item.loader_version ? (
                            <span className="launcher-tag">{item.loader_version}</span>
                          ) : null}
                        </div>
                      </div>
                      <Space wrap className="launcher-instance-actions">
                        <Button
                          type="primary"
                          size="large"
                          icon={<RocketOutlined />}
                          loading={launchingId === item.id}
                          disabled={launchingId !== null && launchingId !== item.id}
                          onClick={() => handlePlay(item)}
                        >
                          {t('launcher.play')}
                        </Button>
                        <Popconfirm
                          title={t('launcher.deleteInstanceConfirm')}
                          onConfirm={() => handleDelete(item.id)}
                        >
                          <Button danger icon={<DeleteOutlined />} />
                        </Popconfirm>
                      </Space>
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>

          <div className="launcher-panel launcher-panel--profiles">
            <div className="launcher-panel-header">
              <div className="launcher-panel-heading">
                <span className="launcher-panel-icon">
                  <UserOutlined />
                </span>
                <div>
                  <Title level={4} className="launcher-panel-title">
                    {t('launcher.offlineProfiles')}
                  </Title>
                  {profiles.length > 0 ? (
                    <Text type="secondary" className="launcher-panel-count">
                      {profiles.length}
                    </Text>
                  ) : null}
                </div>
              </div>
              {canManage && profiles.length > 0 ? (
                <Button icon={<PlusOutlined />} onClick={openProfileModal}>
                  {t('common.add')}
                </Button>
              ) : null}
            </div>

            {!canManage ? (
              <Paragraph type="secondary" className="launcher-panel-empty">
                {t('launcher.offlineAfterLink')}
              </Paragraph>
            ) : profilesLoading ? (
              <div className="launcher-panel-loading">
                <Spin />
              </div>
            ) : profiles.length === 0 ? (
              <div className="launcher-empty">
                <UserOutlined className="launcher-empty-icon" />
                <Paragraph type="secondary">{t('launcher.noProfiles')}</Paragraph>
                <Button type="primary" ghost onClick={openProfileModal}>
                  {t('launcher.addProfile')}
                </Button>
              </div>
            ) : (
              <>
                <Paragraph type="secondary" className="launcher-panel-hint">
                  {t('launcher.selectNickname')}
                </Paragraph>
                <div className="launcher-profile-list">
                  {profiles.map((profile) => {
                    const selected = selectedProfileId === profile.id;
                    return (
                      <div
                        key={profile.id}
                        className={`launcher-profile-chip${selected ? ' launcher-profile-chip--selected' : ''}`}
                      >
                        <button
                          type="button"
                          className="launcher-profile-chip-main"
                          onClick={() => setSelectedProfileId(profile.id)}
                        >
                          <span className="profile-model-chip-avatar" aria-hidden>
                            <ProfileModelAvatar model={profile.model ?? 'steve'} size="sm" />
                          </span>
                          <span className="launcher-profile-chip-text">
                            <span className="launcher-profile-name">
                              {profile.username}
                              {selected ? (
                                <CheckCircleOutlined className="launcher-profile-selected-icon" />
                              ) : null}
                            </span>
                            <span className="launcher-profile-meta">
                              {t(`launcher.profileModel.${profile.model ?? 'steve'}`)} ·{' '}
                              {t(`launcher.profileGender.${profile.model ?? 'steve'}`)}
                            </span>
                          </span>
                        </button>
                        <Popconfirm
                          title={t('launcher.deleteProfileConfirm')}
                          onConfirm={() => handleDeleteProfile(profile.id)}
                        >
                          <Button
                            type="text"
                            danger
                            size="small"
                            icon={<DeleteOutlined />}
                            aria-label={t('launcher.deleteProfileConfirm')}
                          />
                        </Popconfirm>
                      </div>
                    );
                  })}
                </div>
              </>
            )}
          </div>
        </div>
      </section>

      <Modal
        title={t('launcher.newInstance')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
        destroyOnHidden
        {...modalMotionProps}
      >
        <Form
          form={createForm}
          layout="vertical"
          onFinish={handleCreate}
          initialValues={{ loader: DEFAULT_LAUNCHER_LOADER }}
        >
          <Form.Item
            name="name"
            label={t('common.name')}
            rules={[{ required: true, message: t('launcher.nameRequired') }]}
          >
            <Input placeholder="Survival" />
          </Form.Item>
          <Form.Item
            name="loader"
            label={t('launcher.loaderType')}
            rules={[{ required: true, message: t('launcher.loaderTypeRequired') }]}
          >
            <Select options={loaderOptions} onChange={handleCreateLoaderChange} />
          </Form.Item>
          <Form.Item
            name="mc_version"
            label={t('launcher.mcVersion')}
            rules={[{ required: true, message: t('launcher.mcVersionRequired') }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              loading={mcVersionsLoading || createMcOptionsLoading}
              options={
                (createLoader === 'vanilla'
                  ? mcVersionOptions
                  : createMcOptions) as DefaultOptionType[]
              }
              onChange={handleCreateMcVersionChange}
            />
          </Form.Item>
          {createNeedsLoader ? (
            <Form.Item
              name="loader_version"
              label={t(`launcher.loaderVersionLabel.${createLoader}`)}
              rules={[{ required: true, message: t('launcher.loaderVersionRequired') }]}
            >
              <Select
                showSearch
                optionFilterProp="label"
                loading={createLoaderOptionsLoading}
                disabled={createLoaderOptionsLoading || createLoaderOptions.length === 0}
                options={createLoaderOptions}
              />
            </Form.Item>
          ) : null}
          <Button type="primary" htmlType="submit" loading={creating} block>
            {t('launcher.createInstance')}
          </Button>
        </Form>
      </Modal>

      <Modal
        title={t('launcher.newPlayerProfile')}
        open={profileOpen}
        onCancel={() => setProfileOpen(false)}
        footer={null}
        destroyOnHidden
        {...modalMotionProps}
      >
        <Form
          layout="vertical"
          onFinish={handleCreateProfile}
          initialValues={{ model: 'steve' as const }}
        >
          <Form.Item
            name="username"
            label={t('launcher.nickname')}
            rules={[
              { required: true, message: t('launcher.nicknameRequired') },
              { min: 3, message: t('launcher.nicknameMin3') },
              { max: 16, message: t('launcher.nicknameMax16') },
            ]}
          >
            <Input placeholder="Steve" maxLength={16} />
          </Form.Item>
          <Form.Item
            name="model"
            label={t('launcher.profileModelLabel')}
            rules={[{ required: true, message: t('launcher.profileModelRequired') }]}
          >
            <ProfileModelPicker />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={creatingProfile} block>
            {t('common.create')}
          </Button>
        </Form>
      </Modal>
    </div>
  );
}
