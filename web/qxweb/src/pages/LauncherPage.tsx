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
import { logger } from '@/lib/logger';

const LAUNCH_POLL_MS = 1500;
const LAUNCH_TERMINAL = new Set(['completed', 'failed', 'expired']);

function launchStatusMessage(status: string): string {
  switch (status) {
    case 'queued':
      return 'Запрос в очереди…';
    case 'dispatched':
      return 'QXLauncher получил запрос…';
    case 'running':
      return 'Minecraft запускается…';
    case 'completed':
      return 'Игра завершена';
    case 'failed':
      return 'Не удалось запустить игру';
    case 'expired':
      return 'Запрос истёк — проверьте, что QXLauncher запущен';
    /* v8 ignore next 3 */
    default:
      return status;
  }
}

export function LauncherPage() {
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
  const instancesTitle = isAuthenticated ? 'Мои инстансы' : 'Инстансы';

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
      message.error('Не удалось загрузить инстансы');
    } finally {
      setLoading(false);
    }
  }, [canManage]);

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
        /* v8 ignore next 3 */
        if (prev && items.some((p) => p.id === prev)) return prev;
        return items[0]?.id;
      });
    } catch (e) {
      logger.warn('failed to load profiles', { error: String(e) });
      message.error('Не удалось загрузить профили');
    } finally {
      setProfilesLoading(false);
    }
  }, [canManage]);

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
      message.success('Инстанс создан');
      setCreateOpen(false);
      await loadInstances();
      const prof = await api.listProfiles();
      if ((prof.items ?? []).length === 0) {
        message.info('Создайте offline-профиль с ником или играйте с Player по умолчанию');
        setProfileOpen(true);
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Не удалось создать инстанс');
    } finally {
      setCreating(false);
    }
  };

  const handleCreateProfile = async (values: { username: string }) => {
    setCreatingProfile(true);
    try {
      const profile = await api.createProfile({ username: values.username });
      message.success('Профиль создан');
      setProfileOpen(false);
      setProfiles((prev) => [...prev, profile]);
      setSelectedProfileId(profile.id);
    } catch (e) {
      message.error(
        e instanceof Error ? e.message : /* v8 ignore next */ 'Не удалось создать профиль',
      );
    } finally {
      setCreatingProfile(false);
    }
  };

  const handleDeleteProfile = async (id: string) => {
    try {
      await api.deleteProfile(id);
      message.success('Профиль удалён');
      setProfiles((prev) => prev.filter((p) => p.id !== id));
      setSelectedProfileId((prev) => (prev === id ? undefined : /* v8 ignore next */ prev));
    } catch (e) {
      message.error(
        e instanceof Error ? e.message : /* v8 ignore next */ 'Не удалось удалить профиль',
      );
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await api.deleteInstance(id);
      message.success('Инстанс удалён');
      await loadInstances();
    } catch (e) {
      message.error(
        e instanceof Error ? e.message : /* v8 ignore next */ 'Не удалось удалить',
      );
    }
  };

  const pollLaunchRequest = async (requestId: string) => {
    const started = Date.now();
    while (Date.now()-started < 5 * 60 * 1000) {
      const req = await api.getLaunchRequest(requestId);
      message.info(launchStatusMessage(req.status), 2);
      if (LAUNCH_TERMINAL.has(req.status)) {
        if (req.status === 'completed') {
          message.success('Игра запущена');
        } else if (req.status === 'failed') {
          message.error(req.error_code ?? 'Ошибка запуска');
        } else {
          message.warning(launchStatusMessage(req.status));
        }
        return;
      }
      /* v8 ignore next 3 */
      await new Promise((r) => setTimeout(r, LAUNCH_POLL_MS));
    }
    /* v8 ignore next */
    message.warning('Время ожидания истекло');
  };

  const handleUnlinkDevice = async () => {
    try {
      await api.unlinkDevice();
      setLinkedDevice(null);
      clearLinkedDevice();
      message.success('QXLauncher отвязан');
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Не удалось отвязать устройство');
    }
  };

  const handlePlay = async (instance: LauncherInstance) => {
    setLaunchingId(instance.id);
    try {
      if (!selectedProfileId) {
        message.info('Ник Player (по умолчанию). Создайте профиль выше для своего ника.');
      }
      const req = await api.createLaunchRequest({
        instance_id: instance.id,
        offline_profile_id: selectedProfileId,
      });
      message.info('Запрос отправлен в QXLauncher');
      await pollLaunchRequest(req.id);
    } catch (e) {
      message.error(
        e instanceof Error ? e.message : /* v8 ignore next */ 'Не удалось запустить игру',
      );
    } finally {
      setLaunchingId(null);
    }
  };

  return (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 720 }}>
      <Typography.Title level={2}>Лаунчер</Typography.Title>
      <Typography.Paragraph type="secondary">
        Создайте Vanilla-инстанс и offline-профиль на сайте. Запуск — через связанный QXLauncher на ПК.
      </Typography.Paragraph>

      {!authLoading && !canManage && (
        <Alert
          type="info"
          showIcon
          message="Сначала свяжите QXLauncher"
          description={
            <ol style={{ marginBottom: 0, paddingLeft: 20 }}>
              <li>
                Запустите QXLauncher на этом компьютере (
                <Typography.Text code>make launcher</Typography.Text> или{' '}
                <Typography.Text code>bin/qx-launcher.exe</Typography.Text>).
              </li>
              <li>
                В меню QXLauncher выберите «Связать QXLauncher» — откроется страница привязки в
                браузере.
              </li>
              <li>
                Нажмите «Продолжить как гость» (регистрация не нужна) или войдите в аккаунт.
              </li>
              <li>Вернитесь сюда — появятся инстансы и профили.</li>
            </ol>
          }
          action={
            <Space direction="vertical" align="end">
              <Button icon={<ReloadOutlined />} onClick={refreshAccess}>
                Проверить связь
              </Button>
              <Button type="link" size="small" onClick={() => openAuthModal('login')}>
                Войти в аккаунт
              </Button>
            </Space>
          }
        />
      )}

      {isGuestLauncher && (
        <Alert
          type="info"
          showIcon
          message="Гостевой режим"
          description="Vanilla-инстансы привязаны к этому браузеру. Войдите в аккаунт, чтобы сохранить прогресс на всех устройствах."
          action={
            <Button size="small" type="primary" onClick={() => openAuthModal('login')}>
              Войти
            </Button>
          }
        />
      )}

      {isAuthenticated && user && (
        <Alert
          type={linkedDevice ? 'success' : 'warning'}
          showIcon
          message={`Аккаунт ${user.email}`}
          description={
            linkedDevice
              ? `QXLauncher связан (${linkedDevice.device_id}). Инстансы синхронизируются с QXLauncher.`
              : 'QXLauncher ещё не связан с аккаунтом. Запустите QXLauncher и подтвердите привязку на /launcher/link.'
          }
          action={
            linkedDevice ? (
              <Popconfirm title="Отвязать QXLauncher?" onConfirm={() => void handleUnlinkDevice()}>
                <Button size="small" danger>
                  Отвязать
                </Button>
              </Popconfirm>
            ) : undefined
          }
        />
      )}

      <Card title="QXLauncher для ПК">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            Установите QXLauncher, затем в его меню выберите «Связать QXLauncher». Для dev-сборки:{' '}
            <Typography.Text code>make build-launcher</Typography.Text> →{' '}
            <Typography.Text code>bin/qx-launcher.exe</Typography.Text>.
          </Typography.Paragraph>
          <LauncherDownloadButton type="primary" />
        </Space>
      </Card>

      <Card
        title="Offline-профили"
        extra={
          canManage ? (
            <Button icon={<UserOutlined />} onClick={() => setProfileOpen(true)}>
              Добавить
            </Button>
          ) : null
        }
      >
        {!canManage ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            Доступно после связывания QXLauncher — см. инструкцию выше.
          </Typography.Paragraph>
        ) : (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Select
              allowClear
              placeholder="Выберите ник для запуска"
              style={{ width: '100%' }}
              loading={profilesLoading}
              value={selectedProfileId}
              onChange={setSelectedProfileId}
              options={profiles.map((p) => ({ value: p.id, label: p.username }))}
            />
            <List
              loading={profilesLoading}
              locale={{
                emptyText:
                  'Нет профилей — можно играть как Player или добавьте свой ник',
              }}
              dataSource={profiles}
              renderItem={(item) => (
                <List.Item
                  actions={[
                    <Popconfirm
                      key="del"
                      title="Удалить профиль?"
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
              Создать
            </Button>
          ) : null
        }
      >
        {authLoading ? (
          <Spin />
        ) : !canManage ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            После «Продолжить как гость» на странице привязки нажмите «Проверить связь» выше.
          </Typography.Paragraph>
        ) : (
          <List
            loading={loading}
            locale={{ emptyText: 'Пока нет инстансов' }}
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
                    Играть
                  </Button>,
                  <Popconfirm
                    key="del"
                    title="Удалить инстанс?"
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
        title="Новый инстанс"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
        destroyOnHidden
      >
        <Form layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="name"
            label="Название"
            rules={[{ required: true, message: 'Введите название' }]}
          >
            <Input placeholder="Survival" />
          </Form.Item>
          <Form.Item
            name="mc_version"
            label="Версия Minecraft"
            rules={[{ required: true, message: 'Укажите версию' }]}
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
            Создать Vanilla
          </Button>
        </Form>
      </Modal>

      <Modal
        title="Новый offline-профиль"
        open={profileOpen}
        onCancel={() => setProfileOpen(false)}
        footer={null}
        destroyOnHidden
      >
        <Form layout="vertical" onFinish={handleCreateProfile}>
          <Form.Item
            name="username"
            label="Никнейм"
            rules={[
              { required: true, message: 'Введите ник' },
              { min: 3, message: 'Минимум 3 символа' },
              { max: 16, message: 'Максимум 16 символов' },
            ]}
          >
            <Input placeholder="Steve" maxLength={16} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={creatingProfile} block>
            Создать
          </Button>
        </Form>
      </Modal>
    </Space>
  );
}
