import { useState, type DragEvent } from 'react';
import { Link } from 'react-router-dom';
import { Button } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import type { GameServerNetworkMember, GameServerNetworkRole } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { canMoveNetworkMember, groupNetworkMembers } from '@/lib/gameServerNetworkLayout';
import { gameServerTypeLabelText } from '@/lib/gameServerTypes';
import './GameServerNetworkBoard.css';

export function GameServerNetworkBoard({
  vpsId,
  members,
  onMove,
  onRemove,
}: {
  vpsId: string;
  members: GameServerNetworkMember[];
  onMove: (gameServerId: string, role: GameServerNetworkRole) => void;
  onRemove: (gameServerId: string) => void;
}) {
  const { t } = useI18n();
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [overColumn, setOverColumn] = useState<GameServerNetworkRole | null>(null);
  const grouped = groupNetworkMembers(members);
  const dragging = members.find((item) => item.game_server_id === draggingId);

  const columns: Array<{
    id: GameServerNetworkRole;
    title: string;
    items: GameServerNetworkMember[];
  }> = [
    { id: 'proxy', title: t('servers.networkDiagramConnect'), items: grouped.proxy },
    { id: 'lobby', title: t('servers.networkDiagramTry'), items: grouped.lobby },
    { id: 'backend', title: t('servers.networkBoardWorlds'), items: grouped.backend },
  ];

  const finishDrag = () => {
    setDraggingId(null);
    setOverColumn(null);
  };

  return (
    <div className="network-board" role="group" aria-label={t('servers.networksTitle')}>
      {columns.map((column) => {
        const accepts = dragging ? canMoveNetworkMember(dragging.role, column.id) : true;
        const className = [
          'network-board-col',
          `network-board-col--${column.id}`,
          overColumn === column.id && dragging ? 'network-board-col--over' : '',
          overColumn === column.id && dragging && !accepts ? 'network-board-col--blocked' : '',
        ]
          .filter(Boolean)
          .join(' ');
        return (
          <section
            key={column.id}
            className={className}
            data-column={column.id}
            aria-label={column.title}
            onDragOver={(event) => {
              event.preventDefault();
              event.dataTransfer.dropEffect = accepts ? 'move' : 'none';
              setOverColumn(column.id);
            }}
            onDrop={(event) => {
              event.preventDefault();
              const gameServerId =
                draggingId || event.dataTransfer.getData('text/plain') || event.dataTransfer.getData('text');
              finishDrag();
              if (!gameServerId) return;
              const member = members.find((item) => item.game_server_id === gameServerId);
              if (!member || !canMoveNetworkMember(member.role, column.id)) return;
              onMove(gameServerId, column.id);
            }}
          >
            <header className="network-board-col-header">{column.title}</header>
            {column.items.length === 0 ? (
              <p className="network-board-empty">
                {column.id === 'proxy' ? t('servers.networkBoardEmptyProxy') : t('servers.networkBoardDropHere')}
              </p>
            ) : (
              column.items.map((member) => (
                <BoardCard
                  key={member.game_server_id}
                  vpsId={vpsId}
                  member={member}
                  dragging={draggingId === member.game_server_id}
                  onDragStart={(event) => {
                    if (member.role === 'proxy') {
                      event.preventDefault();
                      return;
                    }
                    setDraggingId(member.game_server_id);
                    event.dataTransfer.setData('text/plain', member.game_server_id);
                    event.dataTransfer.setData('text', member.game_server_id);
                    event.dataTransfer.effectAllowed = 'move';
                  }}
                  onDragEnd={finishDrag}
                  onRemove={() => onRemove(member.game_server_id)}
                />
              ))
            )}
          </section>
        );
      })}
    </div>
  );
}

function BoardCard({
  vpsId,
  member,
  dragging,
  onDragStart,
  onDragEnd,
  onRemove,
}: {
  vpsId: string;
  member: GameServerNetworkMember;
  dragging: boolean;
  onDragStart: (event: DragEvent<HTMLElement>) => void;
  onDragEnd: () => void;
  onRemove: () => void;
}) {
  const { t } = useI18n();
  const draggable = member.role !== 'proxy';
  return (
    <article
      className={[
        'network-board-card',
        `network-board-card--${member.role}`,
        draggable ? 'network-board-card--movable' : '',
        dragging ? 'network-board-card--dragging' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      draggable={draggable}
      data-server-id={member.game_server_id}
      aria-grabbed={dragging}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
    >
      <div className="network-board-card-top">
        <Link
          to={`/servers/${vpsId}/game-servers/${member.game_server_id}`}
          className="network-board-card-name"
          draggable={false}
        >
          {member.name || member.alias}
        </Link>
        <Button
          type="text"
          size="small"
          danger
          icon={<DeleteOutlined />}
          aria-label={t('common.delete')}
          draggable={false}
          onClick={onRemove}
        />
      </div>
      <span className="network-board-card-sub">
        {gameServerTypeLabelText(t, member.server_type)}
        {member.port ? ` · :${member.port}` : ''}
      </span>
      {member.in_proxy === false ? (
        <span className="network-board-card-warn">{t('servers.networkMissingInProxy')}</span>
      ) : null}
    </article>
  );
}
