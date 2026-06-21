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

function statusLabel(status: string): string {
  const labels: Record<string, string> = {
    pending: 'Ожидает',
    deploying: 'Deploy…',
    offline: 'Оффлайн',
    starting: 'Запуск…',
    online: 'Онлайн',
    stopping: 'Остановка…',
    error: 'Ошибка',
  };
  return labels[status] ?? status;
}

function ServersList() {
  const navigate = useNavigate();
  const message = useMessage();
  const [servers, setServers] = useState<GameServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listServers();
      setServers(res.items ?? []);
    } catch (e) {
      logger.warn('failed to load servers', { error: String(e) });
      message.error('Не удалось загрузить серверы');
    } finally {
      setLoading(false);
    }
  }, []);

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
      message.success('Сервер добавлен');
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
          Серверы
        </Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          Добавить VPS
        </Button>
      </Space>

      <List
        dataSource={servers}
        locale={{ emptyText: 'Нет серверов — добавьте Linux VPS с SSH-доступом' }}
        renderItem={(item) => (
          <List.Item
            actions={[
              <Link key="open" to={`/servers/${item.id}`}>
                Открыть
              </Link>,
            ]}
          >
            <List.Item.Meta
              avatar={<CloudServerOutlined style={{ fontSize: 24 }} />}
              title={
                <Space>
                  {item.name}
                  <Tag color={statusColor(item.status)}>{statusLabel(item.status)}</Tag>
                  {item.agent_online && <Tag color="blue">Agent</Tag>}
                </Space>
              }
              description={`${item.ssh.host}:${item.ssh.port} · ${item.server_type}`}
            />
          </List.Item>
        )}
      />

      <Modal
        title="Добавить сервер (BYOS)"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void onCreate()}
        confirmLoading={creating}
        width={560}
        destroyOnHidden
        {...modalMotionProps}
      >
        <Form form={form} layout="vertical" initialValues={{ port: 22 }}>
          <Form.Item name="name" label="Название" rules={[{ required: true, message: 'Укажите название' }]}>
            <Input placeholder="Survival VPS" />
          </Form.Item>
          <Form.Item name="mc_version" label="Версия Minecraft">
            <Input placeholder="1.21" />
          </Form.Item>
          <Form.Item name="host" label="SSH Host" rules={[{ required: true }]}>
            <Input placeholder="203.0.113.10" />
          </Form.Item>
          <Form.Item name="port" label="SSH Port">
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="username" label="SSH User" rules={[{ required: true }]}>
            <Input placeholder="root" />
          </Form.Item>
          <Form.Item name="private_key" label="SSH Private Key" rules={[{ required: true }]}>
            <TextArea rows={4} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
          </Form.Item>
          <Form.Item name="jar_path" label="Путь к server.jar на VPS">
            <Input placeholder="/opt/qx/server/server.jar" />
          </Form.Item>
          <Form.Item name="jvm_args" label="JVM args (по одному на строку)">
            <TextArea rows={2} placeholder="-Xmx2G" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

function ServerDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const message = useMessage();
  const [server, setServer] = useState<GameServer | null>(null);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState<string | null>(null);

  const load = useCallback(async () => {
    /* v8 ignore next 3 -- @preserve */
    if (!id) return;
    setLoading(true);
    try {
      const data = await api.getServer(id);
      setServer(data);
    } catch (e) {
      logger.warn('failed to load server', { error: String(e) });
      message.error('Сервер не найден');
      navigate('/servers');
    } finally {
      setLoading(false);
    }
  }, [id, navigate]);

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
        message.success('Deploy выполнен — ожидаем подключение агента');
      } else if (name === 'stop') {
        await api.stopServer(id);
        await load();
      } else {
        await api.restartServer(id);
        await load();
      }
      if (name !== 'deploy') {
        message.success('Готово');
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Ошибка');
    } finally {
      setAction(null);
    }
  };

  const onDelete = async () => {
    /* v8 ignore next 3 -- @preserve */
    if (!id) return;
    try {
      await api.deleteServer(id);
      message.success('Сервер удалён');
      navigate('/servers');
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Ошибка');
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
        <Link to="/servers">← К списку</Link>
      </Space>

      <Card>
        <Space wrap>
          <Tag color={statusColor(server.status)}>{statusLabel(server.status)}</Tag>
          {server.minecraft_running ? (
            <Tag color="green">Minecraft</Tag>
          ) : null}
          {server.agent_online ? (
            <Tag color="blue">Agent подключён</Tag>
          ) : (
            <Tag>Agent оффлайн</Tag>
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

      <Card title="Управление">
        <Space wrap>
          <Button
            icon={<RocketOutlined />}
            loading={busy && action === 'deploy'}
            onClick={() => void runAction('deploy')}
          >
            Deploy agent
          </Button>
          {shouldShowMinecraftControls(server) && (
            <>
              <Button
                loading={busy && action === 'stop'}
                disabled={!server.agent_online}
                onClick={() => void runAction('stop')}
              >
                Stop
              </Button>
              <Button
                icon={<ReloadOutlined />}
                loading={busy && action === 'restart'}
                disabled={!server.agent_online}
                onClick={() => void runAction('restart')}
              >
                Restart
              </Button>
            </>
          )}
          <Popconfirm title="Удалить сервер?" onConfirm={() => void onDelete()}>
            <Button danger icon={<DeleteOutlined />}>
              Удалить
            </Button>
          </Popconfirm>
        </Space>
        {!server.agent_online && (
          <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
            После Deploy агент подключится по WSS автоматически — обычно в течение нескольких секунд.
          </Typography.Paragraph>
        )}
      </Card>

      {shouldShowServerConsole(server) && (
        <Card title="Консоль">
          <ServerConsolePanel serverId={server.id} agentOnline={server.agent_online} />
        </Card>
      )}
    </Space>
  );
}

export function ServersPage() {
  const { isAuthenticated, loading } = useAuth();
  const { openAuthModal } = useAuthModal();

  if (loading) {
    return <Spin />;
  }

  if (!isAuthenticated) {
    return (
      <Card>
        <Typography.Paragraph>Управление серверами доступно после входа.</Typography.Paragraph>
        <Button type="primary" onClick={() => openAuthModal('login')}>
          Войти
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
