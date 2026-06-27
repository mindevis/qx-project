import { useCallback, useEffect, useState } from 'react';
import { Empty, Spin, Table, Typography } from 'antd';
import { api, type GameServerFileEntry } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

type GameServerModsPanelProps = {
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
  supportsMods: boolean;
};

function formatFileSize(size?: number): string {
  if (size == null || size <= 0) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export function GameServerModsPanel({
  vpsId,
  gameServerId,
  agentOnline,
  supportsMods,
}: GameServerModsPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [items, setItems] = useState<GameServerFileEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!agentOnline || !supportsMods) {
      setItems([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await api.listVpsGameServerMods(vpsId, gameServerId);
      setItems((res.items ?? []).filter((item) => !item.dir));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.modsLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [agentOnline, gameServerId, message, supportsMods, t, vpsId]);

  useEffect(() => {
    void load();
  }, [load]);

  if (!supportsMods) {
    return (
      <Typography.Paragraph type="secondary">
        {t('gameServerDetail.modsNotSupported')}
      </Typography.Paragraph>
    );
  }

  if (!agentOnline) {
    return (
      <Typography.Paragraph type="secondary">
        {t('servers.gameServersAgentRequired')}
      </Typography.Paragraph>
    );
  }

  if (loading) {
    return (
      <div className="servers-loading">
        <Spin />
      </div>
    );
  }

  if (items.length === 0) {
    return <Empty description={t('gameServerDetail.modsEmpty')} />;
  }

  return (
    <Table
      className="game-server-mods-table"
      rowKey="path"
      size="small"
      pagination={false}
      dataSource={items}
      columns={[
        { title: t('gameServerDetail.fileName'), dataIndex: 'name', key: 'name' },
        {
          title: t('gameServerDetail.fileSize'),
          key: 'size',
          render: (_, row) => formatFileSize(row.size),
        },
      ]}
    />
  );
}
