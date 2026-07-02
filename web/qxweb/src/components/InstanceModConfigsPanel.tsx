import { useCallback, useEffect, useMemo, useState } from 'react';
import { api, type InstanceResource, type LauncherInstance } from '@/api/client';
import {
  instanceResourceToModConfig,
  ModConfigsByModPanel,
  type ConfigSyncContext,
} from '@/components/ModConfigsByModPanel';

type InstanceModConfigsPanelProps = {
  instance: LauncherInstance;
  deviceLinked: boolean;
};

export function InstanceModConfigsPanel({ instance, deviceLinked }: InstanceModConfigsPanelProps) {
  const [mods, setMods] = useState<InstanceResource[]>([]);

  const loadMods = useCallback(async () => {
    try {
      const res = await api.listInstanceResources(instance.id);
      setMods((res.items ?? []).filter((item) => item.resource_type === 'mod'));
    } catch {
      setMods([]);
    }
  }, [instance.id]);

  useEffect(() => {
    void loadMods();
  }, [loadMods]);

  const modConfigs = useMemo(
    () =>
      mods
        .map(instanceResourceToModConfig)
        .filter((item): item is NonNullable<typeof item> => item != null),
    [mods],
  );

  const configSync: ConfigSyncContext = {
    instanceId: instance.id,
    instanceLoader: instance.loader,
    instanceMcVersion: instance.mc_version,
    deviceLinked,
  };

  const fileApi = useMemo(
    () => ({
      listDir: (path: string) =>
        api.listInstanceFiles(instance.id, path).then((res) => res.items ?? []),
      readFile: (path: string) =>
        api.readInstanceFile(instance.id, path).then((res) => res.content),
      writeFile: (path: string, content: string) =>
        api.writeInstanceFile(instance.id, path, content).then(() => undefined),
    }),
    [instance.id],
  );

  return (
    <ModConfigsByModPanel
      mode="instance"
      available={deviceLinked}
      mods={modConfigs}
      fileApi={fileApi}
      configSync={configSync}
    />
  );
}
