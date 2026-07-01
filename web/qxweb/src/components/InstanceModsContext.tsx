import { createContext, useContext, type ReactNode } from 'react';
import type { LauncherInstance } from '@/api/client';

type InstanceModsContextValue = {
  instance: LauncherInstance;
  canSync: boolean;
  basePath: string;
};

const InstanceModsContext = createContext<InstanceModsContextValue | null>(null);

export function InstanceModsProvider({
  instance,
  canSync,
  children,
}: {
  instance: LauncherInstance;
  canSync: boolean;
  children: ReactNode;
}) {
  const basePath = `/launcher/instances/${instance.id}/resources`;
  return (
    <InstanceModsContext.Provider value={{ instance, canSync, basePath }}>
      {children}
    </InstanceModsContext.Provider>
  );
}

export function useInstanceMods() {
  const ctx = useContext(InstanceModsContext);
  if (!ctx) {
    throw new Error('useInstanceMods must be used within InstanceModsProvider');
  }
  return ctx;
}
