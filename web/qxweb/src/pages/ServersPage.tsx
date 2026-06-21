import { useCallback, useEffect, useState } from 'react';
import { Link, Navigate, Route, Routes, useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import {
  CloudServerOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import { api, type GameServer } from '@/api/client';
import { ServerConsolePanel, shouldShowMinecraftControls, shouldShowServerConsole } from '@/components/ServerConsolePanel';
import { useAuth } from '@/auth/AuthContext';
import { modalMotionProps } from '@/lib/modal';
import { useAuthModal } from '@/auth/AuthModalContext';
import { getServerStatusKey } from '@/i18n';
import { useI18n } from '@/i18n/I18nContext';
import { logger } from '@/lib/logger';
import { useMessage } from '@/hooks/useMessage';

const { TextArea } = Input;

function statusColor(status: string): string {
  switch (status) {
    case 'online':
      return 'success';
    case 'starting':
    case 'deploying':
      return 'processing';
    case 'stopping':
      return 'warning';
    case 'error':
      return 'error';
    default:
      return 'default';
  }
}

function ServersList() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const message = useMessage();
  const [servers, setServers] = useState<GameServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm();

  const statusLabel = (status: string) => {
    const key = getServerStatusKey(status);
    const msg = t(key);
    return msg === key ? status : msg;
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listServers();
      setServers(res.items ?? []);
    } catch (e) {
      logger.warn('failed to load servers', { error: String(e) });
      message.error(t('servers.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const onCreate = async () => {
    try {
      const values = await form.validateFields();
      setCreating(true);
      const server = await api.createServer({
        name: values.name,
        mc_version: values.mc_version,
        ssh: {
          host: values.host,
          port: values.port ?? 22,
          username: values.username,
          private_key: values.private_key,
        },
        config: {
          jar_path: values.jar_path,
          jvm_args: values.jvm_args
            ? values.jvm_args.split('\n').map((s: string) => s.trim()).filter(Boolean)
            : [],
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

  if (loading) {
    return <Spin />;
  }

  return (
    <>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {t('servers.title')}
        </Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('servers.addVps')}
        </Button>
      </Space>

      <List
        dataSource={servers}
        locale={{ emptyText: t('servers.empty') }}
        renderItem={(item) => (
          <List.Item
            actions={[
              <Link key="open" to={`/servers/${item.id}`}>
                {t('common.open')}
              </Link>,
            ]}
          >
            <List.Item.Meta
              avatar={<CloudServerOutlined style={{ fontSize: 24 }} />}
              title={
                <Space>
                  {item.name}
                  <Tag color={statusColor(item.status)}>{statusLabel(item.status)}</Tag>
                  {item.agent_online && <Tag color="blue">{t('common.agent')}</Tag>}
                </Space>
              }
              description={`${item.ssh.host}:${item.ssh.port} · ${item.server_type}`}
            />
          </List.Item>
        )}
      />

      <Modal
        title={t('servers.addByos')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void onCreate()}
        confirmLoading={creating}
        width={560}
        destroyOnHidden
        {...modalMotionProps}
      >
        <Form form={form} layout="vertical" initialValues={{ port: 22 }}>
          <Form.Item
            name="name"
            label={t('common.name')}
            rules={[{ required: true, message: t('servers.nameRequired') }]}
          >
            <Input placeholder="Survival VPS" />
          </Form.Item>
          <Form.Item name="mc_version" label={t('servers.mcVersion')}>
            <Input placeholder="1.21" />
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
          <Form.Item name="private_key" label={t('servers.sshKey')} rules={[{ required: true }]}>
            <TextArea rows={4} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
          </Form.Item>
          <Form.Item name="jar_path" label={t('servers.jarPath')}>
            <Input placeholder="/opt/qx/server/server.jar" />
          </Form.Item>
          <Form.Item name="jvm_args" label={t('servers.jvmArgs')}>
            <TextArea rows={2} placeholder="-Xmx2G" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

function ServerDetail() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const message = useMessage();
  const [server, setServer] = useState<GameServer | null>(null);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState<string | null>(null);

  const statusLabel = (status: string) => {
    const key = getServerStatusKey(status);
    const msg = t(key);
    return msg === key ? status : msg;
  };

  const load = useCallback(async () => {
    /* v8 ignore next 3 -- @preserve */
    if (!id) return;
    setLoading(true);
    try {
      const data = await api.getServer(id);
      setServer(data);
    } catch (e) {
      logger.warn('failed to load server', { error: String(e) });
      message.error(t('servers.notFound'));
      navigate('/servers');
    } finally {
      setLoading(false);
    }
  }, [id, message, navigate, t]);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(timer);
  }, [load]);

  const runAction = async (name: 'deploy' | 'stop' | 'restart') => {
    /* v8 ignore next 3 -- @preserve */
    if (!id) return;
    setAction(name);
    try {
      if (name === 'deploy') {
        const updated = await api.deployServer(id);
        setServer(updated);
        message.success(t('servers.deployDone'));
      } else if (name === 'stop') {
        await api.stopServer(id);
        await load();
      } else {
        await api.restartServer(id);
        await load();
      }
      if (name !== 'deploy') {
        message.success(t('common.done'));
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
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
    return <Spin />;
  }

  const busy = action !== null;

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Space style={{ justifyContent: 'space-between', width: '100%' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {server.name}
        </Typography.Title>
        <Link to="/servers">{t('servers.backToList')}</Link>
      </Space>

      <Card>
        <Space wrap>
          <Tag color={statusColor(server.status)}>{statusLabel(server.status)}</Tag>
          {server.minecraft_running ? (
            <Tag color="green">{t('servers.minecraft')}</Tag>
          ) : null}
          {server.agent_online ? (
            <Tag color="blue">{t('servers.agentOnline')}</Tag>
          ) : (
            <Tag>{t('servers.agentOffline')}</Tag>
          )}
          {server.mc_version && <Tag>MC {server.mc_version}</Tag>}
        </Space>
        <Typography.Paragraph style={{ marginTop: 16 }}>
          SSH: {server.ssh.username}@{server.ssh.host}:{server.ssh.port}
        </Typography.Paragraph>
        {server.config.jar_path && (
          <Typography.Paragraph type="secondary">JAR: {server.config.jar_path}</Typography.Paragraph>
        )}
      </Card>

      <Card title={t('servers.management')}>
        <Space wrap>
          <Button
            icon={<RocketOutlined />}
            loading={busy && action === 'deploy'}
            onClick={() => void runAction('deploy')}
          >
            {t('servers.deployAgent')}
          </Button>
          {shouldShowMinecraftControls(server) && (
            <>
              <Button
                loading={busy && action === 'stop'}
                disabled={!server.agent_online}
                onClick={() => void runAction('stop')}
              >
                {t('servers.stop')}
              </Button>
              <Button
                icon={<ReloadOutlined />}
                loading={busy && action === 'restart'}
                disabled={!server.agent_online}
                onClick={() => void runAction('restart')}
              >
                {t('servers.restart')}
              </Button>
            </>
          )}
          <Popconfirm title={t('servers.deleteConfirm')} onConfirm={() => void onDelete()}>
            <Button danger icon={<DeleteOutlined />}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
        {!server.agent_online && (
          <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
            {t('servers.deployHint')}
          </Typography.Paragraph>
        )}
      </Card>

      {shouldShowServerConsole(server) && (
        <Card title={t('servers.console')}>
          <ServerConsolePanel serverId={server.id} agentOnline={server.agent_online} />
        </Card>
      )}
    </Space>
  );
}

export function ServersPage() {
  const { t } = useI18n();
  const { isAuthenticated, loading } = useAuth();
  const { openAuthModal } = useAuthModal();

  if (loading) {
    return <Spin />;
  }

  if (!isAuthenticated) {
    return (
      <Card>
        <Typography.Paragraph>{t('servers.authRequired')}</Typography.Paragraph>
        <Button type="primary" onClick={() => openAuthModal('login')}>
          {t('auth.signIn')}
        </Button>
      </Card>
    );
  }

  return (
    <Routes>
      <Route index element={<ServersList />} />
      <Route path=":id" element={<ServerDetail />} />
      <Route path="*" element={<Navigate to="/servers" replace />} />
    </Routes>
  );
}
