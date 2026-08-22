import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Empty, Form, Input, Modal, Popconfirm, Select, Space, Typography } from 'antd';
import { ApartmentOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { api, type GameServerNetwork, type GameServerNetworkChange, type GameServerNetworkRole } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { logger } from '@/lib/logger';
import { modalMotionProps } from '@/lib/modal';
import {
  aliasFromServerName,
  suggestedAliasForServer,
  suggestedRoleForServer,
} from '@/lib/gameServerNetworkLayout';
import { gameServerTypeLabelText, isProxyGameServerType, type VpsGameServerType } from '@/lib/gameServerTypes';
import type { VpsGameServer } from '@/lib/vpsGameServers';
import { GameServerNetworkDiagram } from './GameServerNetworkDiagram';

const { Paragraph, Text, Title } = Typography;

type DraftMember = {
  game_server_id: string;
  role: GameServerNetworkRole;
  alias: string;
};

export function GameServerNetworksPanel({
  vpsId,
  games,
  agentOnline,
}: {
  vpsId: string;
  games: VpsGameServer[];
  agentOnline: boolean;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const [networks, setNetworks] = useState<GameServerNetwork[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createName, setCreateName] = useState('');

  const refresh = useCallback(async () => {
    try {
      const res = await api.listVpsGameServerNetworks(vpsId);
      setNetworks(res.items ?? []);
    } catch (e) {
      logger.warn('failed to load game server networks', { error: String(e) });
      message.error(t('servers.networkLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [message, t, vpsId]);

  useEffect(() => {
    void refresh();
  }, [refresh, agentOnline]);

  const assignedIds = useMemo(() => {
    const ids = new Set<string>();
    for (const network of networks) {
      for (const member of network.members ?? []) {
        ids.add(member.game_server_id);
      }
    }
    return ids;
  }, [networks]);

  const createNetwork = async () => {
    const name = createName.trim();
    if (!name) {
      message.error(t('servers.networkNameRequired'));
      return;
    }
    setCreating(true);
    try {
      await api.createVpsGameServerNetwork(vpsId, { name });
      message.success(t('servers.networkCreated'));
      setCreateOpen(false);
      setCreateName('');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('servers.networkSaveFailed'));
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="servers-panel">
      <div className="servers-panel-header">
        <Title level={4} className="servers-panel-title">
          {t('servers.networksTitle')}
        </Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('servers.createNetwork')}
        </Button>
      </div>
      <Paragraph type="secondary" className="servers-hint">
        {t('servers.networksHint')}
      </Paragraph>
      {loading ? null : networks.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <div className="servers-game-empty-copy">
              <Text strong>{t('servers.networksEmpty')}</Text>
              <Paragraph type="secondary">{t('servers.networksEmptyHint')}</Paragraph>
            </div>
          }
        />
      ) : (
        <div className="network-list">
          {networks.map((network) => (
            <NetworkCard
              key={network.id}
              vpsId={vpsId}
              network={network}
              games={games}
              assignedIds={assignedIds}
              agentOnline={agentOnline}
              onChanged={refresh}
            />
          ))}
        </div>
      )}
      <Modal
        title={t('servers.createNetwork')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void createNetwork()}
        confirmLoading={creating}
        okText={t('servers.createNetwork')}
        {...modalMotionProps}
      >
        <Form layout="vertical">
          <Form.Item label={t('servers.networkName')} required>
            <Input
              value={createName}
              maxLength={64}
              placeholder={t('servers.networkNamePlaceholder')}
              onChange={(e) => setCreateName(e.target.value)}
              onPressEnter={() => void createNetwork()}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function NetworkCard({
  vpsId,
  network,
  games,
  assignedIds,
  agentOnline,
  onChanged,
}: {
  vpsId: string;
  network: GameServerNetwork;
  games: VpsGameServer[];
  assignedIds: Set<string>;
  agentOnline: boolean;
  onChanged: () => Promise<void>;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const [name, setName] = useState(network.name);
  const [members, setMembers] = useState<DraftMember[]>(
    (network.members ?? []).map((item) => ({
      game_server_id: item.game_server_id,
      role: item.role,
      alias: item.role === 'proxy' ? 'proxy' : item.alias,
    })),
  );
  const [saving, setSaving] = useState(false);
  const [applying, setApplying] = useState(false);
  const [overwriteOpen, setOverwriteOpen] = useState(false);
  const [overwriteChanges, setOverwriteChanges] = useState<GameServerNetworkChange[]>([]);
  const [overwriteKind, setOverwriteKind] = useState<'save' | 'apply'>('apply');

  useEffect(() => {
    setName(network.name);
    setMembers(
      (network.members ?? []).map((item) => ({
        game_server_id: item.game_server_id,
        role: item.role,
        alias: item.role === 'proxy' ? 'proxy' : item.alias,
      })),
    );
  }, [network]);

  const availableGames = games.filter(
    (game) => !assignedIds.has(game.id) || members.some((member) => member.game_server_id === game.id),
  );
  const unusedGames = availableGames.filter(
    (game) => !members.some((member) => member.game_server_id === game.id),
  );

  const diagramMembers = (network.members ?? []).map((item) => {
    const draft = members.find((member) => member.game_server_id === item.game_server_id);
    return draft ? { ...item, ...draft } : item;
  });
  for (const draft of members) {
    if (diagramMembers.some((item) => item.game_server_id === draft.game_server_id)) continue;
    const game = games.find((item) => item.id === draft.game_server_id);
    if (!game) continue;
    diagramMembers.push({
      id: draft.game_server_id,
      game_server_id: draft.game_server_id,
      role: draft.role,
      alias: draft.alias,
      sort_order: diagramMembers.length,
      name: game.name,
      server_type: game.server_type ?? 'vanilla',
      port: game.port ?? 25565,
      address: game.address,
      status: game.status,
    });
  }

  const save = async (apply: boolean, overwrite = false) => {
    setSaving(true);
    try {
      const updated = await api.updateVpsGameServerNetwork(vpsId, network.id, {
        name: name.trim() || network.name,
        members: members.map((member, index) => {
          const game = games.find((item) => item.id === member.game_server_id);
          const role = member.role;
          return {
            game_server_id: member.game_server_id,
            role,
            alias:
              role === 'proxy'
                ? 'proxy'
                : member.alias.trim() || aliasFromServerName(game?.name ?? member.game_server_id),
            sort_order: index,
          };
        }),
        apply,
        overwrite,
      });
      if (apply && updated.needs_confirm && (updated.proxy_changes?.length ?? 0) > 0) {
        if (updated.applied) {
          message.success(t('servers.networkOverwriteMerged'));
        } else {
          message.success(t('servers.networkUpdated'));
        }
        setOverwriteKind('save');
        setOverwriteChanges(updated.proxy_changes ?? []);
        setOverwriteOpen(true);
        await onChanged();
        return;
      }
      if (apply && updated.applied) {
        message.success(t('servers.networkApplied'));
      } else if (apply && updated.apply_error) {
        message.warning(t('servers.networkApplyPartial', { error: updated.apply_error }));
      } else if (apply && !agentOnline) {
        message.info(t('servers.networkApplyOffline'));
      } else {
        message.success(t('servers.networkUpdated'));
      }
      await onChanged();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('servers.networkSaveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const applyOnly = async (overwrite = false) => {
    setApplying(true);
    try {
      const updated = await api.applyVpsGameServerNetwork(vpsId, network.id, { overwrite });
      if (updated.needs_confirm && (updated.proxy_changes?.length ?? 0) > 0) {
        if (updated.applied) {
          message.success(t('servers.networkOverwriteMerged'));
        }
        setOverwriteKind('apply');
        setOverwriteChanges(updated.proxy_changes ?? []);
        setOverwriteOpen(true);
        return;
      }
      if (updated.applied) {
        message.success(t('servers.networkApplied'));
      } else if (updated.apply_error) {
        message.warning(t('servers.networkApplyPartial', { error: updated.apply_error }));
      } else {
        message.info(t('servers.networkApplyOffline'));
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('servers.networkSaveFailed'));
    } finally {
      setApplying(false);
    }
  };

  const confirmOverwrite = async () => {
    setOverwriteOpen(false);
    if (overwriteKind === 'save') {
      await save(true, true);
      return;
    }
    await applyOnly(true);
  };

  const addServer = (gameId: string) => {
    const game = games.find((item) => item.id === gameId);
    if (!game) return;
    setMembers((prev) => {
      const role = suggestedRoleForServer(game.server_type, prev);
      return [
        ...prev,
        {
          game_server_id: game.id,
          role,
          alias: suggestedAliasForServer(game.name, role),
        },
      ];
    });
  };

  const hasProxy = members.some((member) => member.role === 'proxy');
  const hasLobby = members.some((member) => member.role === 'lobby');

  return (
    <article className="network-card">
      <div className="network-card-header">
        <ApartmentOutlined aria-hidden />
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="network-card-name"
          maxLength={64}
        />
        <Popconfirm title={t('servers.networkDeleteConfirm')} onConfirm={() => {
          void api.deleteVpsGameServerNetwork(vpsId, network.id)
            .then(() => {
              message.success(t('servers.networkDeleted'));
              return onChanged();
            })
            .catch((e: unknown) => message.error(e instanceof Error ? e.message : t('common.error')));
        }}>
          <Button danger icon={<DeleteOutlined />} aria-label={t('common.delete')} />
        </Popconfirm>
      </div>

      {diagramMembers.length > 0 ? (
        <GameServerNetworkDiagram vpsId={vpsId} members={diagramMembers} />
      ) : null}

      {!hasProxy ? (
        <Paragraph type="secondary">{t('servers.networkNeedProxy')}</Paragraph>
      ) : null}
      {hasProxy && !hasLobby ? (
        <Paragraph type="secondary">{t('servers.networkNeedLobby')}</Paragraph>
      ) : null}
      {network.proxy_synced ? (
        <Paragraph type="secondary">{t('servers.networkFromProxy')}</Paragraph>
      ) : null}
      {network.proxy_extra && network.proxy_extra.length > 0 ? (
        <Paragraph type="secondary">
          {t('servers.networkProxyExtra', {
            servers: network.proxy_extra.map((item) => `${item.alias} (${item.address})`).join(', '),
          })}
        </Paragraph>
      ) : null}

      <div className="network-member-list">
        {members.map((member) => {
          const game = games.find((item) => item.id === member.game_server_id);
          return (
            <div key={member.game_server_id} className="network-member-row">
              <Text className="network-member-name">
                {game?.name ?? member.game_server_id}
                <Text type="secondary">
                  {' '}
                  · {gameServerTypeLabelText(t, game?.server_type)}
                </Text>
                {network.proxy_synced &&
                network.members.find((item) => item.game_server_id === member.game_server_id)?.in_proxy ===
                  false ? (
                  <Text type="warning"> · {t('servers.networkMissingInProxy')}</Text>
                ) : null}
              </Text>
              <Select
                value={member.role}
                className="network-member-role"
                options={roleOptions(t, game?.server_type)}
                onChange={(role: GameServerNetworkRole) =>
                  setMembers((prev) =>
                    prev.map((item) =>
                      item.game_server_id === member.game_server_id
                        ? {
                            ...item,
                            role,
                            alias:
                              role === 'proxy'
                                ? 'proxy'
                                : item.role === 'proxy'
                                  ? suggestedAliasForServer(game?.name ?? item.alias, role)
                                  : item.alias,
                          }
                        : item,
                    ),
                  )
                }
              />
              {member.role === 'proxy' ? null : (
                <Input
                  value={member.alias}
                  className="network-member-alias"
                  placeholder={t('servers.networkAlias')}
                  onChange={(e) =>
                    setMembers((prev) =>
                      prev.map((item) =>
                        item.game_server_id === member.game_server_id
                          ? { ...item, alias: e.target.value }
                          : item,
                      ),
                    )
                  }
                />
              )}
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
                onClick={() =>
                  setMembers((prev) => prev.filter((item) => item.game_server_id !== member.game_server_id))
                }
              />
            </div>
          );
        })}
      </div>

      {unusedGames.length > 0 ? (
        <Select
          className="network-add-select"
          placeholder={t('servers.networkAddServer')}
          value={undefined}
          options={unusedGames.map((game) => ({
            value: game.id,
            label: `${game.name} (${gameServerTypeLabelText(t, game.server_type)})`,
          }))}
          onChange={(id) => {
            if (id) addServer(id);
          }}
        />
      ) : games.length === 0 ? (
        <Paragraph type="secondary">{t('servers.networkNoServers')}</Paragraph>
      ) : members.length === games.length ? (
        <Paragraph type="secondary">{t('servers.networkAllAssigned')}</Paragraph>
      ) : null}

      <Paragraph type="secondary" className="servers-hint">
        {t('servers.networkAliasHint')}
      </Paragraph>

      <Space wrap>
        <Button type="primary" loading={saving} onClick={() => void save(true)}>
          {t('common.save')}
        </Button>
        <Button loading={applying} disabled={!agentOnline} onClick={() => void applyOnly()}>
          {t('servers.networkApply')}
        </Button>
      </Space>
      <Modal
        title={t('servers.networkOverwriteTitle')}
        open={overwriteOpen}
        onCancel={() => setOverwriteOpen(false)}
        onOk={() => void confirmOverwrite()}
        okText={t('servers.networkOverwriteConfirm')}
        cancelText={t('servers.networkOverwriteKeep')}
        confirmLoading={saving || applying}
        {...modalMotionProps}
      >
        <Paragraph>{t('servers.networkOverwriteHint')}</Paragraph>
        <ul className="network-overwrite-list">
          {overwriteChanges.map((change, index) => (
            <li key={`${change.field}-${change.name ?? ''}-${index}`}>
              {formatProxyChange(t, change)}
            </li>
          ))}
        </ul>
      </Modal>
    </article>
  );
}

function roleOptions(
  t: (key: string) => string,
  serverType: string | undefined,
): Array<{ value: GameServerNetworkRole; label: string }> {
  const isProxy = serverType ? isProxyGameServerType(serverType as VpsGameServerType) : false;
  if (isProxy) {
    return [{ value: 'proxy', label: t('servers.networkRoleProxy') }];
  }
  return [
    { value: 'lobby', label: t('servers.networkRoleLobby') },
    { value: 'backend', label: t('servers.networkRoleBackend') },
  ];
}

function formatProxyChange(
  t: (key: string, params?: Record<string, string | number>) => string,
  change: GameServerNetworkChange,
): string {
  let field = change.field;
  switch (change.field) {
    case 'bind':
      field = t('servers.networkChangeBind');
      break;
    case 'motd':
      field = t('servers.networkChangeMotd');
      break;
    case 'try':
      field = t('servers.networkChangeTry');
      break;
    case 'forwarding.secret':
      field = t('servers.networkChangeSecret');
      break;
    case 'server':
      field = t('servers.networkChangeServer', { name: change.name ?? '' });
      break;
    case 'remove':
      field = t('servers.networkChangeRemove', { name: change.name ?? '' });
      break;
    default:
      break;
  }
  if (change.field === 'remove' || !change.to) {
    return t('servers.networkChangeLineRemove', { field, from: change.from ?? '' });
  }
  return t('servers.networkChangeLine', { field, from: change.from ?? '', to: change.to ?? '' });
}
