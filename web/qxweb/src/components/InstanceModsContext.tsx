import { createContext, useContext, type ReactNode } from 'react';
import { api, type LauncherInstance } from '@/api/client';
import { ModCatalogProvider, type ModCatalogContextValue } from '@/components/ModCatalogContext';
import { useModInstall } from '@/hooks/useModInstall';
import { isServerManagedInstance } from '@/lib/serverManagedInstance';

type InstanceModsContextValue = {
  instance: LauncherInstance;
  canSync: boolean;
  contentLocked: boolean;
  basePath: string;
};

const InstanceModsContext = createContext<InstanceModsContextValue | null>(null);

function InstanceModsCatalogBridge({
  instance,
  canSync,
  children,
}: {
  instance: LauncherInstance;
  canSync: boolean;
  children: ReactNode;
}) {
  const basePath = `/launcher/instances/${instance.id}/resources`;
  const { installingVersionId, installBatch } = useModInstall(instance.id);
  const contentLocked = isServerManagedInstance(instance);
  const catalog: ModCatalogContextValue = {
    loader: instance.loader,
    mcVersion: instance.mc_version,
    basePath,
    canSyncToServer: canSync && !contentLocked,
    showServerOnlyModal: true,
    instance,
    installingVersionId,
    contentLocked,
    listInstalled: async () => {
      const res = await api.listInstanceResources(instance.id);
      return res.items ?? [];
    },
    installBatch,
    uninstall: async ({ source, projectId, filename, resourceType }) => {
      await api.deleteInstanceResource(instance.id, {
        source,
        project_id: projectId,
        filename,
        resource_type: resourceType,
      });
    },
  };
  return (
    <ModCatalogProvider value={catalog}>
      <InstanceModsContext.Provider value={{ instance, canSync: canSync && !contentLocked, contentLocked, basePath }}>
        {children}
      </InstanceModsContext.Provider>
    </ModCatalogProvider>
  );
}

export function InstanceModsProvider({
  instance,
  canSync,
  children,
}: {
  instance: LauncherInstance;
  canSync: boolean;
  children: ReactNode;
}) {
  return (
    <InstanceModsCatalogBridge instance={instance} canSync={canSync}>
      {children}
    </InstanceModsCatalogBridge>
  );
}

export function useInstanceMods() {
  const ctx = useContext(InstanceModsContext);
  if (!ctx) {
    throw new Error('useInstanceMods must be used within InstanceModsProvider');
  }
  return ctx;
}

export function useInstanceModsOptional() {
  return useContext(InstanceModsContext);
}
