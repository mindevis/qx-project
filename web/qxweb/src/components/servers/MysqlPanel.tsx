import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  Empty,
  Input,
  Popconfirm,
  Radio,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import {
  CloudDownloadOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  api,
  type MysqlGrant,
  type MysqlInstallBody,
  type MysqlStatus,
  type MysqlUser,
  type MysqlView,
} from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { logger } from '@/lib/logger';

const { Paragraph, Text, Title } = Typography;

function mysqlStatusColor(status: MysqlStatus): string {
  switch (status) {
    case 'running':
      return 'success';
    case 'installing':
    case 'starting':
      return 'processing';
    case 'error':
      return 'error';
    case 'installed':
      return 'blue';
    default:
      return 'default';
  }
}

const emptyMysql: MysqlView = {
  status: 'not_installed',
  databases: [],
  users: [],
  privilege_catalog: [],
};

export function MysqlPanel({ vpsId, agentOnline }: { vpsId: string; agentOnline: boolean }) {
  const { t } = useI18n();
  const message = useMessage();
  const [view, setView] = useState<MysqlView>(emptyMysql);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [engine, setEngine] = useState<MysqlInstallBody['engine']>('mariadb');
  const [method, setMethod] = useState<MysqlInstallBody['method']>('docker');
  const [version, setVersion] = useState<MysqlInstallBody['version']>('8.0');
  const [dbName, setDbName] = useState('');
  const [username, setUsername] = useState('');
  const [userHost, setUserHost] = useState('%');
  const [userPassword, setUserPassword] = useState('');
  const [userDatabases, setUserDatabases] = useState<string[]>([]);
  const [userPrivileges, setUserPrivileges] = useState<string[]>(['ALL']);
  const [grantUser, setGrantUser] = useState<MysqlUser | null>(null);
  const [grantDatabase, setGrantDatabase] = useState('');
  const [grantPrivileges, setGrantPrivileges] = useState<string[]>(['ALL']);

  const refresh = useCallback(async () => {
    try {
      const next = await api.getVpsMysql(vpsId);
      setView(next);
    } catch (e) {
      logger.warn('failed to load mysql', { error: String(e) });
      message.error(t('servers.mysqlLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [message, t, vpsId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!agentOnline) return undefined;
    const busyStatus =
      view.status === 'installing' || view.status === 'starting' || view.status === 'stopping';
    if (!busyStatus) return undefined;
    const timer = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(timer);
  }, [agentOnline, refresh, view.status]);

  const run = async (action: () => Promise<MysqlView>, successKey: string) => {
    setBusy(true);
    try {
      const next = await action();
      setView(next);
      message.success(t(successKey));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('servers.mysqlActionFailed'));
    } finally {
      setBusy(false);
    }
  };

  const databases = view.databases ?? [];
  const users = view.users ?? [];
  const catalog =
    view.privilege_catalog && view.privilege_catalog.length > 0
      ? view.privilege_catalog
      : ['ALL', 'SELECT', 'INSERT', 'UPDATE', 'DELETE'];
  const installed = view.status !== 'not_installed';
  const running = view.status === 'running';
  const provisioning =
    view.status === 'installing' || view.status === 'starting' || view.status === 'stopping';
  const allSelected = userPrivileges.includes('ALL');
  const grantAllSelected = grantPrivileges.includes('ALL');

  const versionHint = useMemo(() => {
    if (engine === 'mariadb') {
      return version === '5.7' ? t('servers.mysqlMaria57Hint') : t('servers.mysqlMaria80Hint');
    }
    return version === '5.7' ? t('servers.mysqlPercona57Hint') : t('servers.mysqlPercona80Hint');
  }, [engine, t, version]);

  const togglePrivileges = (next: string[], setter: (value: string[]) => void) => {
    if (next.includes('ALL') && next[next.length - 1] === 'ALL') {
      setter(['ALL']);
      return;
    }
    setter(next.filter((item) => item !== 'ALL'));
  };

  return (
    <div className="servers-panel">
      <div className="servers-panel-header">
        <Title level={4} className="servers-panel-title">
          {t('servers.mysqlTitle')}
        </Title>
        <Space wrap>
          {installed ? (
            <>
              <Tag color={mysqlStatusColor(view.status)}>{t(`servers.mysqlStatus.${view.status}`)}</Tag>
              {view.status !== 'error' ? (
                <Button
                  icon={running ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                  loading={busy || provisioning}
                  disabled={!agentOnline || busy || provisioning}
                  onClick={() =>
                    void run(
                      () => (running ? api.stopVpsMysql(vpsId) : api.startVpsMysql(vpsId)),
                      running ? 'servers.mysqlStopped' : 'servers.mysqlStarted',
                    )
                  }
                >
                  {running ? t('servers.mysqlStop') : t('servers.mysqlStart')}
                </Button>
              ) : null}
              <Popconfirm
                title={t('servers.mysqlUninstallConfirm')}
                onConfirm={() =>
                  void run(() => api.uninstallVpsMysql(vpsId), 'servers.mysqlUninstalled')
                }
              >
                <Button danger icon={<DeleteOutlined />} loading={busy} disabled={busy}>
                  {t('servers.mysqlUninstall')}
                </Button>
              </Popconfirm>
            </>
          ) : (
            <Button
              type="primary"
              icon={<CloudDownloadOutlined />}
              loading={busy || view.status === 'installing'}
              disabled={!agentOnline || busy}
              onClick={() =>
                void run(
                  () => api.installVpsMysql(vpsId, { engine, version, method }),
                  'servers.mysqlInstallDone',
                )
              }
            >
              {t('servers.mysqlInstall')}
            </Button>
          )}
        </Space>
      </div>
      <Paragraph type="secondary" className="servers-hint">
        {t('servers.mysqlHint')}
      </Paragraph>
      {!agentOnline ? (
        <Paragraph type="secondary" className="servers-hint">
          {t('servers.mysqlAgentRequired')}
        </Paragraph>
      ) : loading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : (
        <>
          {!installed ? (
            <div className="mysql-install">
              <div className="mysql-option">
                <Text strong>{t('servers.mysqlEngine')}</Text>
                <Radio.Group
                  value={engine}
                  onChange={(e) => setEngine(e.target.value)}
                  options={[
                    { value: 'mariadb', label: t('servers.mysqlMariaDB') },
                    { value: 'percona', label: t('servers.mysqlPercona') },
                  ]}
                />
              </div>
              <div className="mysql-option">
                <Text strong>{t('servers.mysqlMethod')}</Text>
                <Radio.Group
                  value={method}
                  onChange={(e) => setMethod(e.target.value)}
                  options={[
                    { value: 'docker', label: t('servers.mysqlDocker') },
                    { value: 'native', label: t('servers.mysqlNative') },
                  ]}
                />
              </div>
              <div className="mysql-option">
                <Text strong>{t('servers.mysqlVersion')}</Text>
                <Radio.Group
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                  options={[
                    { value: '5.7', label: '5.7' },
                    { value: '8.0', label: '8.0' },
                  ]}
                />
                <Paragraph type="secondary" className="servers-hint">
                  {versionHint}
                </Paragraph>
              </div>
            </div>
          ) : (
            <dl className="servers-game-card-meta servers-game-card-meta--grid ollama-meta">
              {view.engine ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlEngine')}</dt>
                  <dd>{view.engine === 'percona' ? t('servers.mysqlPercona') : t('servers.mysqlMariaDB')}</dd>
                </div>
              ) : null}
              {view.version ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlVersion')}</dt>
                  <dd>
                    {view.version}
                    {view.package_version && view.package_version !== view.version
                      ? ` (${view.package_version})`
                      : ''}
                  </dd>
                </div>
              ) : null}
              {view.method ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlMethod')}</dt>
                  <dd>{view.method === 'native' ? t('servers.mysqlNative') : t('servers.mysqlDocker')}</dd>
                </div>
              ) : null}
              {view.host_local ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlHostLocal')}</dt>
                  <dd>
                    <Text copyable>{view.host_local}</Text>
                  </dd>
                </div>
              ) : null}
              {view.host_public ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlHostPublic')}</dt>
                  <dd>
                    <Text copyable>{view.host_public}</Text>
                  </dd>
                </div>
              ) : null}
              {view.port ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlPort')}</dt>
                  <dd>
                    <Text copyable={{ text: String(view.port) }}>{view.port}</Text>
                  </dd>
                </div>
              ) : null}
              {view.root_user ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlRootUser')}</dt>
                  <dd>
                    <Text copyable>{view.root_user}</Text>
                  </dd>
                </div>
              ) : null}
              {view.root_password ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlRootPassword')}</dt>
                  <dd>
                    <Text copyable={{ text: view.root_password }} code>
                      {view.root_password}
                    </Text>
                  </dd>
                </div>
              ) : null}
              {view.jdbc ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.mysqlJdbc')}</dt>
                  <dd>
                    <Text copyable code>
                      {view.jdbc}
                    </Text>
                  </dd>
                </div>
              ) : null}
            </dl>
          )}
          {view.last_error ? (
            <Paragraph type="danger" className="servers-hint">
              {view.last_error}
            </Paragraph>
          ) : null}

          {running ? (
            <>
              <div className="mysql-section">
                <Text strong>{t('servers.mysqlDatabases')}</Text>
                <Space.Compact className="mysql-inline">
                  <Input
                    value={dbName}
                    placeholder={t('servers.mysqlDatabasePlaceholder')}
                    onChange={(e) => setDbName(e.target.value)}
                    disabled={busy}
                  />
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    disabled={!dbName.trim() || busy}
                    onClick={() =>
                      void run(
                        () => api.createVpsMysqlDatabase(vpsId, dbName.trim()),
                        'servers.mysqlDatabaseCreated',
                      ).then(() => setDbName(''))
                    }
                  >
                    {t('servers.mysqlCreateDatabase')}
                  </Button>
                </Space.Compact>
                {databases.length === 0 ? (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={t('servers.mysqlDatabasesEmpty')}
                  />
                ) : (
                  <ul className="ollama-model-list">
                    {databases.map((db) => (
                      <li key={db.id} className="ollama-model-row">
                        <Text code>{db.name}</Text>
                        <Popconfirm
                          title={t('servers.mysqlDropDatabaseConfirm', { name: db.name })}
                          onConfirm={() =>
                            void run(
                              () => api.dropVpsMysqlDatabase(vpsId, db.name),
                              'servers.mysqlDatabaseDropped',
                            )
                          }
                        >
                          <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                        </Popconfirm>
                      </li>
                    ))}
                  </ul>
                )}
              </div>

              <div className="mysql-section">
                <Text strong>{t('servers.mysqlUsers')}</Text>
                <div className="mysql-user-form">
                  <Input
                    value={username}
                    placeholder={t('servers.mysqlUsernamePlaceholder')}
                    onChange={(e) => setUsername(e.target.value)}
                    disabled={busy}
                  />
                  <Input
                    value={userHost}
                    placeholder={t('servers.mysqlHostPlaceholder')}
                    onChange={(e) => setUserHost(e.target.value)}
                    disabled={busy}
                  />
                  <Input.Password
                    value={userPassword}
                    placeholder={t('servers.mysqlPasswordPlaceholder')}
                    onChange={(e) => setUserPassword(e.target.value)}
                    disabled={busy}
                  />
                  <Select
                    mode="multiple"
                    allowClear
                    placeholder={t('servers.mysqlBindDatabases')}
                    value={userDatabases}
                    onChange={setUserDatabases}
                    options={databases.map((db) => ({ value: db.name, label: db.name }))}
                    disabled={busy || databases.length === 0}
                  />
                  <Checkbox.Group
                    className="mysql-privs"
                    value={userPrivileges}
                    onChange={(values) => togglePrivileges(values as string[], setUserPrivileges)}
                    options={catalog.map((priv) => ({
                      value: priv,
                      label: priv,
                      disabled: allSelected && priv !== 'ALL',
                    }))}
                  />
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    disabled={!username.trim() || busy || userPrivileges.length === 0}
                    onClick={() => {
                      const grants: MysqlGrant[] = userDatabases.map((database) => ({
                        database,
                        privileges: userPrivileges,
                      }));
                      void run(
                        () =>
                          api.createVpsMysqlUser(vpsId, {
                            username: username.trim(),
                            password: userPassword.trim() || undefined,
                            host: userHost.trim() || '%',
                            grants,
                          }),
                        'servers.mysqlUserCreated',
                      ).then(() => {
                        setUsername('');
                        setUserPassword('');
                        setUserDatabases([]);
                      });
                    }}
                  >
                    {t('servers.mysqlCreateUser')}
                  </Button>
                </div>
                {users.length === 0 ? (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('servers.mysqlUsersEmpty')} />
                ) : (
                  <ul className="ollama-model-list">
                    {users.map((user) => (
                      <li key={user.id} className="mysql-user-card">
                        <div className="mysql-user-card-head">
                          <div>
                            <Text strong>
                              {user.username}@{user.host}
                            </Text>
                            <div>
                              <Text type="secondary">{t('servers.mysqlPassword')}: </Text>
                              <Text copyable={{ text: user.password }} code>
                                {user.password}
                              </Text>
                            </div>
                            {user.dsn ? (
                              <div>
                                <Text type="secondary">{t('servers.mysqlDsn')}: </Text>
                                <Text copyable code>
                                  {user.dsn}
                                </Text>
                              </div>
                            ) : null}
                          </div>
                          <Popconfirm
                            title={t('servers.mysqlDropUserConfirm', { name: `${user.username}@${user.host}` })}
                            onConfirm={() =>
                              void run(
                                () => api.dropVpsMysqlUser(vpsId, user.username, user.host),
                                'servers.mysqlUserDropped',
                              )
                            }
                          >
                            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                          </Popconfirm>
                        </div>
                        <div className="mysql-grants">
                          {(user.grants ?? []).length === 0 ? (
                            <Text type="secondary">{t('servers.mysqlNoGrants')}</Text>
                          ) : (
                            user.grants.map((grant) => (
                              <Tag key={grant.database}>
                                {grant.database}: {grant.privileges.join(', ')}
                              </Tag>
                            ))
                          )}
                        </div>
                        {grantUser?.id === user.id ? (
                          <div className="mysql-user-form">
                            <Select
                              placeholder={t('servers.mysqlBindDatabases')}
                              value={grantDatabase || undefined}
                              onChange={setGrantDatabase}
                              options={databases.map((db) => ({ value: db.name, label: db.name }))}
                            />
                            <Checkbox.Group
                              className="mysql-privs"
                              value={grantPrivileges}
                              onChange={(values) =>
                                togglePrivileges(values as string[], setGrantPrivileges)
                              }
                              options={catalog.map((priv) => ({
                                value: priv,
                                label: priv,
                                disabled: grantAllSelected && priv !== 'ALL',
                              }))}
                            />
                            <Space>
                              <Button
                                type="primary"
                                disabled={!grantDatabase || grantPrivileges.length === 0 || busy}
                                onClick={() => {
                                  const nextGrants = (user.grants ?? []).filter((item) => item.database !== grantDatabase);
                                  nextGrants.push({ database: grantDatabase, privileges: grantPrivileges });
                                  void run(
                                    () =>
                                      api.setVpsMysqlUserGrants(vpsId, user.username, user.host, nextGrants),
                                    'servers.mysqlGrantsSaved',
                                  ).then(() => setGrantUser(null));
                                }}
                              >
                                {t('servers.mysqlSaveGrants')}
                              </Button>
                              <Button onClick={() => setGrantUser(null)}>{t('common.cancel')}</Button>
                            </Space>
                          </div>
                        ) : (
                          <Button size="small" onClick={() => {
                            setGrantUser(user);
                            setGrantDatabase(user.grants[0]?.database ?? databases[0]?.name ?? '');
                            setGrantPrivileges(user.grants[0]?.privileges ?? ['ALL']);
                          }}>
                            {t('servers.mysqlEditGrants')}
                          </Button>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </>
          ) : null}
        </>
      )}
    </div>
  );
}
