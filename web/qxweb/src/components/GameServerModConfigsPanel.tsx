import { useCallback, useEffect, useMemo, useState } from 'react';
import { api, loadLinkedDevice, type GameServerFileEntry } from '@/api/client';
import { ModConfigsByModPanel, serverModToModConfig, type ConfigSyncContext } from '@/components/ModConfigsByModPanel';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

type GameServerModConfigsPanelProps = {
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
  mcVersion?: string;
  loader?: string;
};

export function GameServerModConfigsPanel({
  vpsId,
  gameServerId,
  agentOnline,
  mcVersion,
  loader,
}: GameServerModConfigsPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [serverMods, setServerMods] = useState<GameServerFileEntry[]>([]);
  const [boundInstanceId, setBoundInstanceId] = useState<string | undefined>();
  const deviceLinked = loadLinkedDevice() != null;

  const loadMods = useCallback(async () => {
    if (!agentOnline) {
      setServerMods([]);
      return;
    }
    try {
      const [modsRes, clientRes] = await Promise.all([
        api.listVpsGameServerMods(vpsId, gameServerId),
        api.listVpsGameServerClientMods(vpsId, gameServerId).catch(() => ({ items: [] })),
      ]);
      setServerMods(
        [...(modsRes.items ?? []), ...(clientRes.items ?? [])].filter((item) => !item.dir),
      );
    } catch (e) {
      setServerMods([]);
      message.error(e instanceof Error ? e.message : t('gameServerDetail.content.mod.loadFailed'));
    }
  }, [agentOnline, gameServerId, message, t, vpsId]);

  const loadBinding = useCallback(async () => {
    try {
      const res = await api.listMonitoringBindings();
      const binding = (res.items ?? []).find((item) => item.game_server_id === gameServerId);
      setBoundInstanceId(binding?.instance_id);
    } catch {
      setBoundInstanceId(undefined);
    }
  }, [gameServerId]);

  useEffect(() => {
    void loadMods();
  }, [loadMods]);

  useEffect(() => {
    void loadBinding();
  }, [loadBinding]);

  const modConfigs = useMemo(
    () => (serverMods ?? []).map(serverModToModConfig),
    [serverMods],
  );

  const configSync: ConfigSyncContext | undefined = boundInstanceId
    ? {
        instanceId: boundInstanceId,
        instanceLoader: loader ?? 'fabric',
        instanceMcVersion: mcVersion,
        deviceLinked,
        vpsId,
        gameServerId,
        agentOnline,
      }
    : undefined;

  const fileApi = useMemo(
    () => ({
      listDir: (path: string) =>
        api.listVpsGameServerFiles(vpsId, gameServerId, path).then((res) => res.items ?? []),
      readFile: (path: string) =>
        api.readVpsGameServerFile(vpsId, gameServerId, path).then((res) => res.content),
      writeFile: (path: string, content: string) =>
        api.writeVpsGameServerFile(vpsId, gameServerId, path, content).then(() => undefined),
    }),
    [gameServerId, vpsId],
  );

  return (
    <ModConfigsByModPanel
      mode="server"
      available={agentOnline}
      mods={modConfigs}
      fileApi={fileApi}
      configSync={configSync}
    />
  );
}
