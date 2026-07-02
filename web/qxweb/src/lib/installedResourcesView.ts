import { useCallback, useState } from 'react';

export type InstalledResourcesViewMode = 'list' | 'cards';

export const INSTALLED_RESOURCES_VIEW_STORAGE_KEY = 'qxweb-installed-resources-view';

export function readInstalledResourcesViewMode(): InstalledResourcesViewMode {
  /* v8 ignore next 3 -- @preserve */
  if (typeof window === 'undefined') {
    return 'list';
  }

  const stored = window.localStorage.getItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY);
  if (stored === 'list' || stored === 'cards') {
    return stored;
  }

  return 'list';
}

export function useInstalledResourcesViewMode() {
  const [viewMode, setViewModeState] = useState<InstalledResourcesViewMode>(readInstalledResourcesViewMode);

  const setViewMode = useCallback((nextMode: InstalledResourcesViewMode) => {
    setViewModeState(nextMode);
    window.localStorage.setItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY, nextMode);
  }, []);

  return { viewMode, setViewMode };
}
