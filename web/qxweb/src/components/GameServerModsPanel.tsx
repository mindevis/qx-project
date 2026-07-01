import { GameServerContentPanel } from '@/components/GameServerContentPanel';
import type { VpsGameServerType } from '@/lib/gameServerTypes';

type GameServerModsPanelProps = {
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
  supportsMods: boolean;
  serverType?: VpsGameServerType;
  mcVersion?: string;
};

export function GameServerModsPanel({
  vpsId,
  gameServerId,
  agentOnline,
  supportsMods,
  serverType = 'forge',
  mcVersion = '1.21',
}: GameServerModsPanelProps) {
  return (
    <GameServerContentPanel
      kind="mod"
      vpsId={vpsId}
      gameServerId={gameServerId}
      agentOnline={agentOnline}
      supported={supportsMods}
      serverType={serverType}
      mcVersion={mcVersion}
    />
  );
}
