import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, Navigate, Route, Routes } from 'react-router-dom';
import {
  Alert,
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Segmented,
  Select,
  Space,
  Spin,
  Tabs,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import {
  CheckCircleOutlined,
  DeleteOutlined,
  DesktopOutlined,
  DownloadOutlined,
  AppstoreOutlined,
  CloudDownloadOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  GlobalOutlined,
  LinkOutlined,
  LoginOutlined,
  PlusOutlined,
  RocketOutlined,
  ReloadOutlined,
  SettingOutlined,
  SkinOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { LauncherDownloadButton } from '@/components/LauncherDownloadButton';
import { LauncherCodeSigningNotice } from '@/components/LauncherCodeSigningNotice';
import { InstanceServerBinding } from '@/components/InstanceServerBinding';
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
  ApiRequestError,
  type LauncherInstance,
  type MojangLinkStatus,
  type OfflineProfile,
} from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { getLaunchStatusKey } from '@/i18n';
import { useI18n } from '@/i18n/I18nContext';
import { modalMotionProps } from '@/lib/modal';
import { logger } from '@/lib/logger';
import { cachedListMcVersions } from '@/lib/mcVersionsCache';
import { isUpdateAvailable } from '@/lib/launcherVersion';
import { openLauncherDownload, resolveLauncherDownloadUrl, type LauncherRelease } from '@/lib/launcherDownload';
import {
  launcherSupportsModsCatalog,
  launcherSupportsResourcesPage,
} from '@/lib/launcherInstanceCapabilities';
import { InstanceFilesPanel } from '@/components/InstanceFilesPanel';
import { InstanceOptionsPanel } from '@/components/InstanceOptionsPanel';
import { InstanceModConfigsPanel } from '@/components/InstanceModConfigsPanel';
import {
  getLaunchErrorKey,
  isLaunchTerminal,
  type LaunchProgressState,
} from '@/lib/launchProgress';
import {
  isPrepareActive,
  isPrepareTerminal,
  type PrepareProgressState,
} from '@/lib/prepareProgress';
import { LauncherInstanceResourcesPage } from '@/pages/LauncherInstanceResourcesPage';
import './LauncherPage.css';

const { Title, Paragraph, Text } = Typography;

const LAUNCH_POLL_MS = 1500;

type LaunchAccountMode = 'offline' | 'licensed';

export function LauncherPage() {
  return (
    <Routes>
      <Route index element={<LauncherHome />} />
      <Route path="instances/:instanceId/resources/*" element={<LauncherInstanceResourcesPage />} />
      <Route path="*" element={<Navigate to="/launcher" replace />} />
    </Routes>
  );
}

const defaultInstanceMemoryMb = 4096;

function LauncherHome() {
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
  const [launchProgress, setLaunchProgress] = useState<LaunchProgressState | null>(null);
  const [prepareProgress, setPrepareProgress] = useState<Record<string, PrepareProgressState>>({});
  const [linkedDevice, setLinkedDevice] = useState<{
    device_id: string;
    status: string;
    launcher_version?: string;
  } | null>(null);
  const [launcherRelease, setLauncherRelease] = useState<LauncherRelease | null>(null);
  const [updateRequesting, setUpdateRequesting] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [accountMode, setAccountMode] = useState<LaunchAccountMode>('offline');
  const [mojangStatus, setMojangStatus] = useState<MojangLinkStatus | null>(null);
  const [mojangLoading, setMojangLoading] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsInstance, setSettingsInstance] = useState<LauncherInstance | null>(null);
  const [settingsTab, setSettingsTab] = useState<'launch' | 'options' | 'files' | 'mods'>('launch');
  const [settingsName, setSettingsName] = useState('');
  const [settingsRamMb, setSettingsRamMb] = useState(defaultInstanceMemoryMb);
  const [settingsMinRamMb, setSettingsMinRamMb] = useState(defaultInstanceMemoryMb);
  const [settingsExtraJvmArgs, setSettingsExtraJvmArgs] = useState('');
  const [settingsWindowWidth, setSettingsWindowWidth] = useState<number | null>(null);
  const [settingsWindowHeight, setSettingsWindowHeight] = useState<number | null>(null);
  const [savingSettings, setSavingSettings] = useState(false);
  const userChoseAccountMode = useRef(false);
  const [, setAccessKey] = useState(0);
  const refreshAccess = useCallback(() => setAccessKey((k) => k + 1), []);
  const canManage = !authLoading && isAuthenticated;
  const instancesTitle = t('launcher.myInstances');
  const selectedProfile = profiles.find((p) => p.id === selectedProfileId);
  const licensedReady = accountMode === 'licensed' && mojangStatus?.linked === true;
  const sortedInstances = [...instances].sort((a, b) =>
    a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }),
  );
  const launchBlocked = accountMode === 'licensed' && !licensedReady;
  const updateAvailable =
    linkedDevice != null &&
    launcherRelease != null &&
    isUpdateAvailable(linkedDevice.launcher_version, launcherRelease.version);
  const downloadUrl = resolveLauncherDownloadUrl(launcherRelease);
  const activePlayerLabel =
    accountMode === 'licensed'
      ? mojangStatus?.linked
        ? (mojangStatus.username ?? t('launcher.licensedAccount'))
        : t('launcher.licensedNotLinked')
      : (selectedProfile?.username ?? t('launcher.playerDefault'));

  const accountModeOptions = [
    { label: t('launcher.accountModeOffline'), value: 'offline' as const },
    { label: t('launcher.accountModeLicensed'), value: 'licensed' as const },
  ];

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
        /* v8 ignore next 4 -- @preserve upstream mc version list may fail */
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
        /* v8 ignore next 4 -- @preserve upstream loader version list may fail */
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
      const result = await cachedListMcVersions();
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

  const launchErrorMessage = useCallback(
    (errorCode?: string) => {
      if (!errorCode) {
        return t('launcher.launchError');
      }
      const key = getLaunchErrorKey(errorCode);
      if (!key) {
        return errorCode;
      }
      const msg = t(key);
      return msg === key ? errorCode : msg;
    },
    [t],
  );

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

  const loadMojangStatus = useCallback(async () => {
    if (!canManage) {
      setMojangStatus(null);
      return;
    }
    setMojangLoading(true);
    try {
      const status = await api.mojangStatus();
      setMojangStatus(status);
      if (!userChoseAccountMode.current) {
        setAccountMode(status.linked ? 'licensed' : 'offline');
      } else if (!status.linked) {
        setAccountMode('offline');
      }
    } catch (e) {
      /* v8 ignore next 3 -- @preserve mojang status is optional for offline play */
      logger.warn('failed to load mojang status', { error: String(e) });
      setMojangStatus(null);
    } finally {
      setMojangLoading(false);
    }
  }, [canManage]);

  const refreshWorkspace = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([loadInstances(), loadProfiles(), loadMojangStatus()]);
      message.success(t('launcher.workspaceRefreshed'));
    } finally {
      setRefreshing(false);
    }
  }, [loadInstances, loadMojangStatus, loadProfiles, message, t]);

  useEffect(() => {
    void loadInstances();
    void loadProfiles();
    void loadMojangStatus();
  }, [loadInstances, loadMojangStatus, loadProfiles]);

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
    void (async () => {
      try {
        const release = await api.getLauncherRelease();
        setLauncherRelease(release);
      } catch (e) {
        logger.warn('failed to load launcher release', { error: String(e) });
      }
    })();
  }, []);

  useEffect(() => {
    if (!isAuthenticated) {
      setLinkedDevice(null);
      return;
    }
    (async () => {
      try {
        const res = await api.myLauncherDevice();
        if (res.linked && res.device_id) {
          setLinkedDevice({
            device_id: res.device_id,
            status: res.status ?? 'linked',
            launcher_version: res.launcher_version,
          });
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
      const created = await api.createInstance({
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
      if (created.prepare_request_id) {
        void pollPrepareRequest(created.prepare_request_id, created.id);
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

  const pollPrepareRequest = async (requestId: string, instanceId: string) => {
    const started = Date.now();
    const timeoutMs = 30 * 60 * 1000;
    while (Date.now() - started < timeoutMs) {
      const req = await api.getPrepareRequest(requestId);
      setPrepareProgress((prev) => ({
        ...prev,
        [instanceId]: {
          instanceId,
          requestId,
          status: req.status,
          errorCode: req.error_code,
        },
      }));
      if (isPrepareTerminal(req.status)) {
        return;
      }
      /* v8 ignore next 3 -- @preserve */
      await new Promise((r) => setTimeout(r, LAUNCH_POLL_MS));
    }
    setPrepareProgress((prev) => ({
      ...prev,
      [instanceId]: {
        instanceId,
        requestId,
        status: 'failed',
        errorCode: 'PREPARE_TIMEOUT',
      },
    }));
  };

  const pollLaunchRequest = async (
    requestId: string,
    instanceId: string,
    launchAccountMode: LaunchAccountMode,
  ) => {
    const started = Date.now();
    const timeoutMs = 30 * 60 * 1000;
    while (Date.now() - started < timeoutMs) {
      try {
        const req = await api.getLaunchRequest(requestId);
        setLaunchProgress({
          instanceId,
          requestId,
          status: req.status,
          accountMode: launchAccountMode,
          errorCode: req.error_code,
          needsMojangRelink:
            launchAccountMode === 'licensed' && req.error_code === 'MOJANG_SESSION',
        });
        if (isLaunchTerminal(req.status)) {
          setLaunchProgress((prev) =>
            prev?.requestId === requestId
              ? {
                  ...prev,
                  status: req.status,
                  errorCode: req.error_code,
                  needsMojangRelink:
                    launchAccountMode === 'licensed' && req.error_code === 'MOJANG_SESSION',
                }
              : prev,
          );
          return;
        }
      } catch (e) {
        if (e instanceof ApiRequestError && e.apiCode === 'MOJANG_UNAVAILABLE') {
          continue;
        }
        if (e instanceof ApiRequestError && e.apiCode === 'MOJANG_SESSION') {
          setLaunchProgress({
            instanceId,
            requestId,
            status: 'failed',
            accountMode: launchAccountMode,
            errorCode: 'MOJANG_SESSION',
            needsMojangRelink: launchAccountMode === 'licensed',
          });
          return;
        }
        throw e;
      }
      /* v8 ignore next 3 -- @preserve */
      await new Promise((r) => setTimeout(r, LAUNCH_POLL_MS));
    }
    /* v8 ignore next -- @preserve */
    setLaunchProgress({
      instanceId,
      requestId,
      status: 'failed',
      accountMode: launchAccountMode,
      errorCode: 'LAUNCH_TIMEOUT',
    });
  };

  const handleRequestLauncherUpdate = async () => {
    setUpdateRequesting(true);
    try {
      await api.requestLauncherUpdate();
      message.success(t('launcher.updateRequested'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('launcher.updateRequestFailed'));
    } finally {
      setUpdateRequesting(false);
    }
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

  const openInstanceSettings = (instance: LauncherInstance) => {
    setSettingsInstance(instance);
    setSettingsTab('launch');
    setSettingsName(instance.name);
    setSettingsRamMb(instance.max_memory_mb ?? defaultInstanceMemoryMb);
    setSettingsMinRamMb(instance.min_memory_mb ?? defaultInstanceMemoryMb);
    setSettingsExtraJvmArgs((instance.extra_jvm_args ?? []).join('\n'));
    setSettingsWindowWidth(instance.window_width ?? null);
    setSettingsWindowHeight(instance.window_height ?? null);
    setSettingsOpen(true);
  };

  const handleSaveInstanceSettings = async () => {
    if (!settingsInstance) return;
    setSavingSettings(true);
    try {
      const extraJvmArgs = settingsExtraJvmArgs
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean);
      const updated = await api.updateInstance(settingsInstance.id, {
        name: settingsName.trim(),
        max_memory_mb: settingsRamMb,
        min_memory_mb: settingsMinRamMb,
        extra_jvm_args: extraJvmArgs,
        window_width: settingsWindowWidth ?? 0,
        window_height: settingsWindowHeight ?? 0,
      });
      setInstances((prev) => prev.map((i) => (i.id === updated.id ? updated : i)));
      message.success(t('launcher.instanceSettingsSaved'));
      setSettingsOpen(false);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('launcher.instanceSettingsFailed'));
    } finally {
      setSavingSettings(false);
    }
  };

  const handlePlay = async (instance: LauncherInstance) => {
    if (accountMode === 'licensed' && !licensedReady) {
      message.warning(t('launcher.licensedLaunchFailed'));
      return;
    }
    const launchAccountMode = accountMode;
    setLaunchProgress({
      instanceId: instance.id,
      requestId: '',
      status: 'queued',
      accountMode: launchAccountMode,
    });
    try {
      if (launchAccountMode === 'offline' && !selectedProfileId) {
        message.info(t('launcher.defaultPlayerHint'));
      }
      const req = await api.createLaunchRequest({
        instance_id: instance.id,
        offline_profile_id: launchAccountMode === 'offline' ? selectedProfileId : undefined,
        use_mojang_account: launchAccountMode === 'licensed',
      });
      setLaunchProgress({
        instanceId: instance.id,
        requestId: req.id,
        status: req.status,
        accountMode: launchAccountMode,
      });
      await pollLaunchRequest(req.id, instance.id, launchAccountMode);
    } catch (e) {
      setLaunchProgress(null);
      if (e instanceof Error) {
        message.error(e.message);
      } else {
        message.error(t('launcher.launchGameFailed'));
      }
    } finally {
      window.setTimeout(() => {
        setLaunchProgress((prev) =>
          prev?.instanceId === instance.id && isLaunchTerminal(prev.status) ? null : prev,
        );
      }, 8000);
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
              <Space wrap size="middle" className="launcher-hero-actions">
                <Button
                  type="primary"
                  size="large"
                  icon={<PlusOutlined />}
                  disabled={!canManage}
                  onClick={openCreateModal}
                >
                  {t('launcher.heroCreateInstance')}
                </Button>
                <Button size="large" icon={<UserOutlined />} onClick={openProfileModal}>
                  {t('launcher.heroAddProfile')}
                </Button>
              </Space>
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

        {isAuthenticated ? (
          <nav className="launcher-ecosystem-links" aria-label={t('launcher.ecosystemNav')}>
            <Link to="/monitoring" className="launcher-ecosystem-link">
              <GlobalOutlined aria-hidden />
              {t('layout.navMonitoring')}
            </Link>
            <Link to="/skins" className="launcher-ecosystem-link">
              <SkinOutlined aria-hidden />
              {t('layout.navSkins')}
            </Link>
            <Link to="/servers" className="launcher-ecosystem-link">
              <CloudServerOutlined aria-hidden />
              {t('layout.navServers')}
            </Link>
          </nav>
        ) : null}

        <section className="launcher-download-public" aria-label={t('launcher.downloadSectionAria')}>
          <div className="launcher-download-band launcher-download-band--public">
            <div className="launcher-download-band-icon">
              <DownloadOutlined />
            </div>
            <div className="launcher-download-band-text">
              <Title level={5} className="launcher-download-band-title">
                {t('launcher.desktopTitle')}
              </Title>
              <Paragraph className="launcher-download-band-desc">
                {isAuthenticated ? t('launcher.desktopDesc') : t('launcher.downloadPublicDesc')}
              </Paragraph>
              <LauncherCodeSigningNotice />
            </div>
            <LauncherDownloadButton type="primary" release={launcherRelease} />
          </div>
        </section>

        {linkedDevice && isAuthenticated && updateAvailable && launcherRelease ? (
          <Alert
            type="info"
            showIcon
            className="launcher-update-alert"
            title={t('launcher.updateAvailableTitle')}
            description={t('launcher.updateAvailableDesc', {
              installed: linkedDevice.launcher_version?.trim() || t('launcherLink.unknown'),
              latest: launcherRelease.version,
            })}
            action={
              <Space wrap>
                <Button
                  type="primary"
                  icon={<CloudDownloadOutlined />}
                  loading={updateRequesting}
                  onClick={() => void handleRequestLauncherUpdate()}
                >
                  {t('launcher.updateButton')}
                </Button>
                <Button
                  icon={<DownloadOutlined />}
                  onClick={() => openLauncherDownload(downloadUrl)}
                >
                  {t('launcher.downloadLatest')}
                </Button>
              </Space>
            }
          />
        ) : null}

        {linkedDevice && isAuthenticated && !updateAvailable ? (
          <div className="launcher-download-band launcher-download-band--compact">
            <div className="launcher-download-band-text">
              <Paragraph className="launcher-download-band-desc">
                {linkedDevice.launcher_version
                  ? t('launcher.installedVersion', { version: linkedDevice.launcher_version })
                  : null}
                {launcherRelease
                  ? ` · ${t('launcher.latestVersion', { version: launcherRelease.version })}`
                  : null}
              </Paragraph>
            </div>
            <Button icon={<DownloadOutlined />} onClick={() => openLauncherDownload(downloadUrl)}>
              {t('launcher.downloadLatest')}
            </Button>
          </div>
        ) : null}

        {linkedDevice && isAuthenticated && (
          <div className="launcher-qxmods-promo">
            <Title level={4} className="launcher-qxmods-promo-title">
              {t('qxmods.promoTitle')}
            </Title>
            <Paragraph type="secondary" className="launcher-qxmods-promo-body">
              {t('qxmods.promoBody')}
            </Paragraph>
          </div>
        )}

        <div className="launcher-workspace-stack">
          {canManage ? (
            <div className="launcher-panel launcher-panel--player">
              <div className="launcher-panel-header">
                <div className="launcher-panel-heading">
                  <span className="launcher-panel-icon">
                    <UserOutlined />
                  </span>
                  <div>
                    <Title level={4} className="launcher-panel-title">
                      {t('launcher.playerSectionTitle')}
                    </Title>
                    <Text type="secondary" className="launcher-panel-count">
                      {t('launcher.playerSectionHint')}
                    </Text>
                  </div>
                </div>
                {accountMode === 'offline' ? (
                  <Button icon={<PlusOutlined />} onClick={openProfileModal}>
                    {t('launcher.addProfile')}
                  </Button>
                ) : null}
              </div>

              <div className="launcher-player-controls">
                <div className="launcher-launch-bar-mode">
                  <Text type="secondary">{t('launcher.accountMode')}</Text>
                  <Segmented<LaunchAccountMode>
                    options={accountModeOptions}
                    value={accountMode}
                    onChange={(value) => {
                      userChoseAccountMode.current = true;
                      setAccountMode(value);
                    }}
                  />
                </div>
                <div className="launcher-launch-bar">
                  <Text type="secondary">{t('launcher.playingAs')}</Text>
                  <Text strong className="launcher-launch-bar-name">
                    {mojangLoading && accountMode === 'licensed' ? '…' : activePlayerLabel}
                  </Text>
                  {accountMode === 'offline' && selectedProfile ? (
                    <ProfileModelAvatar model={selectedProfile.model ?? 'steve'} size="sm" />
                  ) : null}
                </div>
              </div>

              {accountMode === 'licensed' ? (
                mojangLoading ? (
                  <div className="launcher-panel-loading">
                    <Spin />
                  </div>
                ) : mojangStatus?.linked ? (
                  <div className="launcher-licensed-account-card">
                    <Text type="secondary" className="launcher-panel-hint">
                      {t('launcher.accountModeLicensedDefault')}
                    </Text>
                    <Text strong>{mojangStatus.username}</Text>
                    {mojangStatus.minecraft_uuid ? (
                      <Text type="secondary" className="launcher-licensed-account-uuid">
                        {mojangStatus.minecraft_uuid}
                      </Text>
                    ) : null}
                  </div>
                ) : (
                  <div className="launcher-empty launcher-empty--inline">
                    <LinkOutlined className="launcher-empty-icon" />
                    <Paragraph type="secondary">{t('launcher.licensedNotLinked')}</Paragraph>
                    <Link to="/profile">
                      <Button type="primary" icon={<LinkOutlined />}>
                        {t('launcher.linkMicrosoft')}
                      </Button>
                    </Link>
                  </div>
                )
              ) : profilesLoading ? (
                <div className="launcher-panel-loading">
                  <Spin />
                </div>
              ) : profiles.length === 0 ? (
                <div className="launcher-empty launcher-empty--inline">
                  <UserOutlined className="launcher-empty-icon" />
                  <Paragraph type="secondary">{t('launcher.noProfiles')}</Paragraph>
                  <Paragraph type="secondary" className="launcher-empty-hint">
                    {t('launcher.noProfilesHint')}
                  </Paragraph>
                  <Button type="primary" icon={<PlusOutlined />} onClick={openProfileModal}>
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
                            onClick={() => {
                              userChoseAccountMode.current = true;
                              setAccountMode('offline');
                              setSelectedProfileId(profile.id);
                            }}
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
          ) : null}

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
                      {t('launcher.instancesSorted', { count: instances.length })}
                    </Text>
                  ) : null}
                </div>
              </div>
              {canManage ? (
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
                  {t('launcher.heroCreateInstance')}
                </Button>
              ) : null}
            </div>

            {!linkedDevice && isAuthenticated && canManage ? (
              <Alert
                type="warning"
                showIcon
                className="launcher-instance-link-alert"
                title={t('launcher.linkDevicePrompt')}
                action={
                  <Link to="/launcher/link">
                    <Button size="small" type="primary" icon={<LinkOutlined />}>
                      {t('launcher.openLinkPage')}
                    </Button>
                  </Link>
                }
              />
            ) : null}

            {launchProgress && !isLaunchTerminal(launchProgress.status) ? (
              <Alert
                type="info"
                showIcon
                className="launcher-launch-progress-alert"
                title={t('launcher.launchProgressTitle')}
                description={launchStatusMessage(launchProgress.status)}
              />
            ) : null}

            {launchProgress && launchProgress.status === 'failed' ? (
              <Alert
                type="error"
                showIcon
                className="launcher-launch-progress-alert"
                title={t('launcher.launchFailedTitle')}
                description={launchErrorMessage(launchProgress.errorCode)}
                action={
                  launchProgress.needsMojangRelink ? (
                    <Link to="/profile">
                      <Button size="small" type="primary" icon={<LinkOutlined />}>
                        {t('launcher.launchRelinkMicrosoft')}
                      </Button>
                    </Link>
                  ) : undefined
                }
              />
            ) : null}

            {authLoading ? (
              <div className="launcher-panel-loading">
                <Spin />
              </div>
            ) : !canManage ? (
              <div className="launcher-empty">
                <LoginOutlined className="launcher-empty-icon" />
                <Paragraph type="secondary" className="launcher-panel-empty">
                  {t('launcher.signInRequired')}
                </Paragraph>
              </div>
            ) : loading ? (
              <div className="launcher-panel-loading">
                <Spin />
              </div>
            ) : instances.length === 0 ? (
              <div className="launcher-empty">
                <RocketOutlined className="launcher-empty-icon" />
                <Paragraph strong>{t('launcher.noInstances')}</Paragraph>
                <Paragraph type="secondary" className="launcher-empty-hint">
                  {t('launcher.noInstancesHint')}
                </Paragraph>
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
                  {t('launcher.createFirstInstance')}
                </Button>
              </div>
            ) : (
              <div className="launcher-instance-list">
                {sortedInstances.map((item) => {
                  const progress =
                    launchProgress?.instanceId === item.id ? launchProgress : null;
                  const prepare =
                    prepareProgress[item.id] &&
                    isPrepareActive(prepareProgress[item.id].status)
                      ? prepareProgress[item.id]
                      : null;
                  const prepareFailed =
                    prepareProgress[item.id]?.status === 'failed'
                      ? prepareProgress[item.id]
                      : null;
                  const isLaunching = progress != null && !isLaunchTerminal(progress.status);
                  const isInstalling = prepare != null;
                  const launchFailed = progress?.status === 'failed';
                  const showResources = launcherSupportsResourcesPage(item.loader);
                  const resourcesLabel = launcherSupportsModsCatalog(item.loader)
                    ? t('launcher.browseResources')
                    : t('launcher.browseDatapacks');
                  const ResourcesIcon = launcherSupportsModsCatalog(item.loader)
                    ? AppstoreOutlined
                    : DatabaseOutlined;
                  return (
                    <div
                      key={item.id}
                      className={`launcher-instance-card${isLaunching ? ' launcher-instance-card--launching' : ''}${isInstalling ? ' launcher-instance-card--launching' : ''}${launchFailed || prepareFailed ? ' launcher-instance-card--failed' : ''}`}
                    >
                      <div className="launcher-instance-info">
                        <div className="launcher-instance-name-row">
                          <Text strong className="launcher-instance-name">
                            {item.name}
                          </Text>
                          {progress ? (
                            <span
                              className={`launcher-instance-status${isLaunching ? ' launcher-instance-status--active' : ''}${launchFailed ? ' launcher-instance-status--failed' : ''}`}
                            >
                              {isLaunching ? (
                                <>
                                  <span>{t('launcher.launching')}</span>
                                  <span className="launcher-instance-status-step">
                                    {launchStatusMessage(progress.status)}
                                  </span>
                                </>
                              ) : launchFailed ? (
                                <span className="launcher-instance-status-step">
                                  {launchErrorMessage(progress.errorCode)}
                                </span>
                              ) : null}
                            </span>
                          ) : prepare ? (
                            <span className="launcher-instance-status launcher-instance-status--active">
                              <span>{t('launcher.installing')}</span>
                              <span className="launcher-instance-status-step">
                                {launchStatusMessage(prepare.status)}
                              </span>
                            </span>
                          ) : prepareFailed ? (
                            <span className="launcher-instance-status launcher-instance-status--failed">
                              <span className="launcher-instance-status-step">
                                {launchErrorMessage(prepareFailed.errorCode)}
                              </span>
                            </span>
                          ) : null}
                        </div>
                        <div className="launcher-instance-tags">
                          <span className="launcher-tag launcher-tag--version">
                            {t('launcher.minecraftVersionPrefix')} {item.mc_version}
                          </span>
                          <span className="launcher-tag">
                            {isLauncherLoader(item.loader) ? loaderLabel(item.loader) : item.loader}
                          </span>
                          {item.loader_version ? (
                            <span className="launcher-tag">{item.loader_version}</span>
                          ) : null}
                        </div>
                        {launchFailed ? (
                          <Paragraph type="danger" className="launcher-instance-launch-error">
                            {launchErrorMessage(progress?.errorCode)}
                          </Paragraph>
                        ) : prepareFailed ? (
                          <Paragraph type="danger" className="launcher-instance-launch-error">
                            {launchErrorMessage(prepareFailed.errorCode)}
                          </Paragraph>
                        ) : null}
                        <InstanceServerBinding instance={item} variant="card" />
                      </div>
                      <Space wrap className="launcher-instance-actions">
                        <Button
                          type="primary"
                          size="large"
                          icon={<RocketOutlined />}
                          loading={isLaunching || isInstalling}
                          disabled={
                            launchBlocked ||
                            isInstalling ||
                            (launchProgress != null &&
                              launchProgress.instanceId !== item.id &&
                              !isLaunchTerminal(launchProgress.status))
                          }
                          onClick={() => handlePlay(item)}
                        >
                          {t('launcher.play')}
                        </Button>
                        {launchFailed && progress?.needsMojangRelink ? (
                          <Link to="/profile">
                            <Button size="large" icon={<LinkOutlined />}>
                              {t('launcher.launchRelinkMicrosoft')}
                            </Button>
                          </Link>
                        ) : null}
                        {showResources ? (
                          <Link to={`/launcher/instances/${item.id}/resources`}>
                            <Button size="large" icon={<ResourcesIcon />}>
                              {resourcesLabel}
                            </Button>
                          </Link>
                        ) : null}
                        <Button
                          size="large"
                          icon={<SettingOutlined />}
                          aria-label={t('launcher.instanceSettings')}
                          onClick={() => openInstanceSettings(item)}
                        />
                        <Popconfirm
                          title={t('launcher.deleteInstanceConfirm')}
                          onConfirm={() => handleDelete(item.id)}
                        >
                          <Button
                            danger
                            icon={<DeleteOutlined />}
                            aria-label={t('launcher.deleteInstanceConfirm')}
                          />
                        </Popconfirm>
                      </Space>
                    </div>
                  );
                })}
              </div>
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
            <Input placeholder={t('launcher.placeholderInstanceName')} />
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
            <Input placeholder={t('launcher.placeholderNickname')} maxLength={16} />
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

      <Modal
        title={t('launcher.instanceSettingsTitle', { name: settingsInstance?.name ?? '' })}
        open={settingsOpen}
        onCancel={() => setSettingsOpen(false)}
        onOk={() => void handleSaveInstanceSettings()}
        confirmLoading={savingSettings}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        destroyOnHidden
        width={640}
        {...modalMotionProps}
      >
        <Tabs
          activeKey={settingsTab}
          onChange={(key) => setSettingsTab(key as 'launch' | 'options' | 'files' | 'mods')}
          items={[
            {
              key: 'launch',
              label: t('launcher.instanceSettingsTabLaunch'),
              children: (
                <>
                  <Paragraph type="secondary">{t('launcher.instanceSettingsHint')}</Paragraph>
                  <Form layout="vertical">
                    <Form.Item label={t('common.name')}>
                      <Input
                        value={settingsName}
                        maxLength={128}
                        onChange={(e) => setSettingsName(e.target.value)}
                        placeholder={t('launcher.placeholderInstanceName')}
                      />
                    </Form.Item>
                    <Form.Item label={t('launcher.minMemoryMb')}>
                      <InputNumber
                        min={512}
                        max={65536}
                        step={512}
                        addonAfter={t('common.megabytes')}
                        value={settingsMinRamMb}
                        onChange={(value) => setSettingsMinRamMb(value ?? defaultInstanceMemoryMb)}
                        style={{ width: '100%' }}
                        placeholder="—"
                      />
                    </Form.Item>
                    <Form.Item label={t('launcher.maxMemoryMb')}>
                      <InputNumber
                        min={512}
                        max={65536}
                        step={512}
                        addonAfter={t('common.megabytes')}
                        value={settingsRamMb}
                        onChange={(value) => setSettingsRamMb(value ?? defaultInstanceMemoryMb)}
                        style={{ width: '100%' }}
                      />
                    </Form.Item>
                    <Form.Item
                      label={t('launcher.extraJvmArgs')}
                      extra={t('launcher.extraJvmArgsHint')}
                    >
                      <Input.TextArea
                        rows={4}
                        value={settingsExtraJvmArgs}
                        onChange={(e) => setSettingsExtraJvmArgs(e.target.value)}
                        placeholder="-XX:+UseG1GC"
                      />
                    </Form.Item>
                    <Space size="middle" style={{ width: '100%' }}>
                      <Form.Item label={t('launcher.windowWidth')} style={{ flex: 1, marginBottom: 0 }}>
                        <InputNumber
                          min={320}
                          max={7680}
                          value={settingsWindowWidth}
                          onChange={(value) => setSettingsWindowWidth(value)}
                          style={{ width: '100%' }}
                          placeholder="—"
                        />
                      </Form.Item>
                      <Form.Item label={t('launcher.windowHeight')} style={{ flex: 1, marginBottom: 0 }}>
                        <InputNumber
                          min={320}
                          max={7680}
                          value={settingsWindowHeight}
                          onChange={(value) => setSettingsWindowHeight(value)}
                          style={{ width: '100%' }}
                          placeholder="—"
                        />
                      </Form.Item>
                    </Space>
                  </Form>
                </>
              ),
            },
            {
              key: 'options',
              label: t('launcher.instanceSettingsTabOptions'),
              children: settingsInstance ? (
                <InstanceOptionsPanel
                  instanceId={settingsInstance.id}
                  deviceLinked={linkedDevice != null}
                />
              ) : null,
            },
            {
              key: 'files',
              label: t('launcher.instanceSettingsTabFiles'),
              children: settingsInstance ? (
                <InstanceFilesPanel
                  instanceId={settingsInstance.id}
                  deviceLinked={linkedDevice != null}
                />
              ) : null,
            },
            {
              key: 'mods',
              label: t('launcher.instanceSettingsTabMods'),
              children: settingsInstance ? (
                <InstanceModConfigsPanel
                  instance={settingsInstance}
                  deviceLinked={linkedDevice != null}
                />
              ) : null,
            },
          ]}
        />
      </Modal>
    </div>
  );
}
