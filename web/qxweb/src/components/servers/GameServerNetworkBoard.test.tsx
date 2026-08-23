import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/test-utils';
import type { GameServerNetworkMember } from '@/api/client';
import { GameServerNetworkBoard } from './GameServerNetworkBoard';

function member(
  partial: Partial<GameServerNetworkMember> & Pick<GameServerNetworkMember, 'game_server_id' | 'role' | 'alias'>,
): GameServerNetworkMember {
  return {
    id: partial.id ?? partial.game_server_id,
    sort_order: partial.sort_order ?? 0,
    name: partial.name ?? partial.alias,
    server_type: partial.server_type ?? 'paper',
    port: partial.port ?? 25565,
    status: partial.status ?? 'stopped',
    ...partial,
  };
}

function dataTransfer() {
  const store: Record<string, string> = {};
  return {
    store,
    setData: (key: string, value: string) => {
      store[key] = value;
    },
    getData: (key: string) => store[key] ?? '',
    effectAllowed: 'all',
    dropEffect: 'move',
  };
}

describe('GameServerNetworkBoard', () => {
  const members = [
    member({
      game_server_id: 'v',
      role: 'proxy',
      alias: 'proxy',
      name: 'Velocity',
      server_type: 'velocity',
    }),
    member({
      game_server_id: 'l',
      role: 'lobby',
      alias: 'lobby',
      name: 'Lobby',
    }),
    member({
      game_server_id: 's',
      role: 'backend',
      alias: 'survival',
      name: 'Survival',
    }),
  ];

  it('shows proxy and try columns without a /server transfer label', () => {
    renderWithProviders(
      <GameServerNetworkBoard vpsId="srv-1" members={members} onMove={vi.fn()} onRemove={vi.fn()} />,
    );
    expect(screen.getByRole('group', { name: 'Проекты серверов' })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'прокси' })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'вход (try)' })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Игровые миры' })).toBeInTheDocument();
    expect(screen.queryByText(/перевод/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/\/server/i)).not.toBeInTheDocument();
    expect(screen.getByText('Velocity')).toBeInTheDocument();
    expect(screen.getByText('Lobby')).toBeInTheDocument();
    expect(screen.getByText('Survival')).toBeInTheDocument();
  });

  it('moves a world card onto the try column', () => {
    const onMove = vi.fn();
    renderWithProviders(
      <GameServerNetworkBoard vpsId="srv-1" members={members} onMove={onMove} onRemove={vi.fn()} />,
    );
    const transfer = dataTransfer();
    const card = screen.getByText('Survival').closest('article');
    const tryColumn = screen.getByRole('region', { name: 'вход (try)' });
    expect(card).toBeTruthy();
    fireEvent.dragStart(card!, { dataTransfer: transfer });
    fireEvent.dragOver(tryColumn, { dataTransfer: transfer });
    fireEvent.drop(tryColumn, { dataTransfer: transfer });
    expect(onMove).toHaveBeenCalledWith('s', 'lobby');
  });

  it('does not move a proxy card onto try', () => {
    const onMove = vi.fn();
    renderWithProviders(
      <GameServerNetworkBoard vpsId="srv-1" members={members} onMove={onMove} onRemove={vi.fn()} />,
    );
    const transfer = dataTransfer();
    const card = screen.getByText('Velocity').closest('article');
    const tryColumn = screen.getByRole('region', { name: 'вход (try)' });
    fireEvent.dragStart(card!, { dataTransfer: transfer });
    fireEvent.drop(tryColumn, { dataTransfer: transfer });
    expect(onMove).not.toHaveBeenCalled();
  });
});
