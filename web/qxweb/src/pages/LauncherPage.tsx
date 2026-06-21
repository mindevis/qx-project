import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  List,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Typography,
} from 'antd';
import {
  DeleteOutlined,
  PlusOutlined,
  RocketOutlined,
  ReloadOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { LauncherDownloadButton } from '@/components/LauncherDownloadButton';
import { useMessage } from '@/hooks/useMessage';
import { DEFAULT_MC_VERSION, MVP_MC_VERSIONS } from '@/launcher/mcVersions';
import {
  api,
  clearLinkedDevice,
  hasLauncherAccess,
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
  const [profileOpen, setProfileOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [creatingProfile, setCreatingProfile] = useState(false);
  const [launchingId, setLaunchingId] = useState<string | null>(null);
  const [linkedDevice, setLinkedDevice] = useState<{ device_id: string; status: string } | null>(
    null,
  );
  const [, setAccessKey] = useState(0);
  const refreshAccess = useCallback(() => setAccessKey((k) => k + 1), []);
  const canManage = !authLoading && (isAuthenticated || hasLauncherAccess());
  const isGuestLauncher = !isAuthenticated && hasLauncherAccess();
  const instancesTitle = isAuthenticated ? t('launcher.myInstances') : t('launcher.instances');

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

  useEffect(() => {
    void loadInstances();
    void loadProfiles();
  }, [loadInstances, loadProfiles]);

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

  const handleCreate = async (values: { name: string; mc_version: string }) => {
    setCreating(true);
    try {
      await api.createInstance({
        name: values.name,
        mc_version: values.mc_version,
        loader: 'vanilla',
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

  const handleCreateProfile = async (values: { username: string }) => {
    setCreatingProfile(true);
    try {
      const profile = await api.createProfile({ username: values.username });
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
    while (Date.now() - started < 5 * 60 * 1000) {
      const req = await api.getLaunchRequest(requestId);
      message.info(launchStatusMessage(req.status), 2);
      if (LAUNCH_TERMINAL.has(req.status)) {
        if (req.status === 'completed') {
          message.success(t('launcher.gameLaunched'));
        } else if (req.status === 'failed') {
          message.error(req.error_code ?? t('launcher.launchError'));
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
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 720 }}>
      <Typography.Title level={2}>{t('launcher.title')}</Typography.Title>
      <Typography.Paragraph type="secondary">{t('launcher.intro')}</Typography.Paragraph>

      {!authLoading && !canManage && (
        <Alert
          type="info"
          showIcon
          message={t('launcher.linkFirstTitle')}
          description={
            <ol style={{ marginBottom: 0, paddingLeft: 20 }}>
              <li>
                {t('launcher.linkStep1')}
                <Typography.Text code>make launcher</Typography.Text>
                {t('launcher.linkStep1Or')}
                <Typography.Text code>bin/qx-launcher.exe</Typography.Text>
                {t('launcher.linkStep1End')}
              </li>
              <li>{t('launcher.linkStep2')}</li>
              <li>{t('launcher.linkStep3')}</li>
              <li>{t('launcher.linkStep4')}</li>
            </ol>
          }
          action={
            <Space direction="vertical" align="end">
              <Button icon={<ReloadOutlined />} onClick={refreshAccess}>
                {t('launcher.checkLink')}
              </Button>
              <Button type="link" size="small" onClick={() => openAuthModal('login')}>
                {t('launcher.signInToAccount')}
              </Button>
            </Space>
          }
        />
      )}

      {isGuestLauncher && (
        <Alert
          type="info"
          showIcon
          message={t('launcher.guestModeTitle')}
          description={t('launcher.guestModeDesc')}
          action={
            <Button size="small" type="primary" onClick={() => openAuthModal('login')}>
              {t('auth.signIn')}
            </Button>
          }
        />
      )}

      {isAuthenticated && user && (
        <Alert
          type={linkedDevice ? 'success' : 'warning'}
          showIcon
          message={t('launcher.accountLabel', { email: user.email })}
          description={
            linkedDevice
              ? t('launcher.linkedDesc', { deviceId: linkedDevice.device_id })
              : t('launcher.notLinkedDesc')
          }
          action={
            linkedDevice ? (
              <Popconfirm
                title={t('launcher.unlinkConfirm')}
                onConfirm={() => void handleUnlinkDevice()}
              >
                <Button size="small" danger>
                  {t('launcher.unlink')}
                </Button>
              </Popconfirm>
            ) : undefined
          }
        />
      )}

      <Card title={t('launcher.desktopTitle')}>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            {t('launcher.desktopDesc')}{' '}
            <Typography.Text code>make build-launcher</Typography.Text>
            {t('launcher.desktopDescArrow')}
            <Typography.Text code>bin/qx-launcher.exe</Typography.Text>.
          </Typography.Paragraph>
          <LauncherDownloadButton type="primary" />
        </Space>
      </Card>

      <Card
        title={t('launcher.offlineProfiles')}
        extra={
          canManage ? (
            <Button icon={<UserOutlined />} onClick={() => setProfileOpen(true)}>
              {t('common.add')}
            </Button>
          ) : null
        }
      >
        {!canManage ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t('launcher.offlineAfterLink')}
          </Typography.Paragraph>
        ) : (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Select
              allowClear
              placeholder={t('launcher.selectNickname')}
              style={{ width: '100%' }}
              loading={profilesLoading}
              value={selectedProfileId}
              onChange={setSelectedProfileId}
              options={profiles.map((p) => ({ value: p.id, label: p.username }))}
            />
            <List
              loading={profilesLoading}
              locale={{
                emptyText: t('launcher.noProfiles'),
              }}
              dataSource={profiles}
              renderItem={(item) => (
                <List.Item
                  actions={[
                    <Popconfirm
                      key="del"
                      title={t('launcher.deleteProfileConfirm')}
                      onConfirm={() => handleDeleteProfile(item.id)}
                    >
                      <Button danger icon={<DeleteOutlined />} />
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta title={item.username} description={item.offline_uuid} />
                </List.Item>
              )}
            />
          </Space>
        )}
      </Card>

      <Card
        title={instancesTitle}
        extra={
          canManage ? (
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
              {t('common.create')}
            </Button>
          ) : null
        }
      >
        {authLoading ? (
          <Spin />
        ) : !canManage ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t('launcher.afterGuestLink')}
          </Typography.Paragraph>
        ) : (
          <List
            loading={loading}
            locale={{ emptyText: t('launcher.noInstances') }}
            dataSource={instances}
            renderItem={(item) => (
              <List.Item
                actions={[
                  <Button
                    key="play"
                    type="primary"
                    icon={<RocketOutlined />}
                    loading={launchingId === item.id}
                    disabled={launchingId !== null && launchingId !== item.id}
                    onClick={() => handlePlay(item)}
                  >
                    {t('launcher.play')}
                  </Button>,
                  <Popconfirm
                    key="del"
                    title={t('launcher.deleteInstanceConfirm')}
                    onConfirm={() => handleDelete(item.id)}
                  >
                    <Button danger icon={<DeleteOutlined />} />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  title={item.name}
                  description={`${item.loader} · Minecraft ${item.mc_version}`}
                />
              </List.Item>
            )}
          />
        )}
      </Card>

      <Modal
        title={t('launcher.newInstance')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
        destroyOnHidden
        {...modalMotionProps}
      >
        <Form layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="name"
            label={t('common.name')}
            rules={[{ required: true, message: t('launcher.nameRequired') }]}
          >
            <Input placeholder="Survival" />
          </Form.Item>
          <Form.Item
            name="mc_version"
            label={t('launcher.mcVersion')}
            rules={[{ required: true, message: t('launcher.mcVersionRequired') }]}
            initialValue={DEFAULT_MC_VERSION}
          >
            <Select
              options={MVP_MC_VERSIONS.map((version) => ({
                value: version,
                label: version,
              }))}
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={creating} block>
            {t('launcher.createVanilla')}
          </Button>
        </Form>
      </Modal>

      <Modal
        title={t('launcher.newOfflineProfile')}
        open={profileOpen}
        onCancel={() => setProfileOpen(false)}
        footer={null}
        destroyOnHidden
        {...modalMotionProps}
      >
        <Form layout="vertical" onFinish={handleCreateProfile}>
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
          <Button type="primary" htmlType="submit" loading={creatingProfile} block>
            {t('common.create')}
          </Button>
        </Form>
      </Modal>
    </Space>
  );
}
