import { useState } from 'react';
import { Segmented } from 'antd';
import { GameServerContentPanel } from '@/components/GameServerContentPanel';
import type { GameServerContentKind } from '@/api/client';
import { gameServerCatalogTabs, type VpsGameServerType } from '@/lib/gameServerTypes';
import { useI18n } from '@/i18n/I18nContext';
import './InstanceResourcesPanel.css';

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
  const { t } = useI18n();
  const tabTypes = gameServerCatalogTabs(serverType);
  const [kind, setKind] = useState<GameServerContentKind>(() => tabTypes[0] ?? 'mod');

  return (
    <div className="game-server-mods-catalog">
      {tabTypes.length > 1 ? (
        <Segmented
          className="qxmods-type-segmented"
          value={kind}
          options={tabTypes.map((type) => ({ value: type, label: t(`qxmods.tabs.${type}`) }))}
          onChange={(value) => setKind(value as GameServerContentKind)}
        />
      ) : null}
      <GameServerContentPanel
        key={kind}
        kind={kind}
        vpsId={vpsId}
        gameServerId={gameServerId}
        agentOnline={agentOnline}
        supported={kind === 'mod' ? supportsMods : true}
        serverType={serverType}
        mcVersion={mcVersion}
      />
    </div>
  );
}
