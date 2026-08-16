import { createContext, useContext, type ReactNode } from 'react';
import type { InstanceResource, LauncherInstance, ModProjectType, ModSource } from '@/api/client';
import type { ModInstallParams } from '@/hooks/useModInstall';

export type ModCatalogUninstallArgs = {
  source: ModSource;
  projectId: string;
  filename?: string;
  resourceType: ModProjectType;
};

export type ModCatalogContextValue = {
  loader: string;
  mcVersion: string;
  basePath: string;
  canSyncToServer: boolean;
  showServerOnlyModal: boolean;
  instance?: LauncherInstance;
  installingVersionId?: string;
  listInstalled: () => Promise<InstanceResource[]>;
  installBatch: (items: ModInstallParams[]) => Promise<boolean>;
  uninstall: (args: ModCatalogUninstallArgs) => Promise<void>;
  contentLocked?: boolean;
};

const ModCatalogContext = createContext<ModCatalogContextValue | null>(null);

export function ModCatalogProvider({
  value,
  children,
}: {
  value: ModCatalogContextValue;
  children: ReactNode;
}) {
  return <ModCatalogContext.Provider value={value}>{children}</ModCatalogContext.Provider>;
}

export function useModCatalog() {
  const ctx = useContext(ModCatalogContext);
  if (!ctx) {
    throw new Error('useModCatalog must be used within ModCatalogProvider');
  }
  return ctx;
}

export function useModCatalogOptional() {
  return useContext(ModCatalogContext);
}
