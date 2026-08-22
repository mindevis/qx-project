import { useCallback, useState } from 'react';

export type InstalledResourcesViewMode = 'list' | 'cards';

export const INSTALLED_RESOURCES_VIEW_STORAGE_KEY = 'qxweb-installed-resources-view';
export const GAME_SERVER_CONTENT_VIEW_STORAGE_KEY = 'qxweb-game-server-content-view';
export const GAME_SERVERS_LIST_VIEW_STORAGE_KEY = 'qxweb-game-servers-list-view';

function readViewMode(
  storageKey: string,
  fallback: InstalledResourcesViewMode,
): InstalledResourcesViewMode {
  /* v8 ignore next 3 -- @preserve */
  if (typeof window === 'undefined') {
    return fallback;
  }

  const stored = window.localStorage.getItem(storageKey);
  if (stored === 'list' || stored === 'cards') {
    return stored;
  }

  return fallback;
}

function useStoredViewMode(storageKey: string, fallback: InstalledResourcesViewMode) {
  const [viewMode, setViewModeState] = useState<InstalledResourcesViewMode>(() =>
    readViewMode(storageKey, fallback),
  );

  const setViewMode = useCallback(
    (nextMode: InstalledResourcesViewMode) => {
      setViewModeState(nextMode);
      window.localStorage.setItem(storageKey, nextMode);
    },
    [storageKey],
  );

  return { viewMode, setViewMode };
}

export function readInstalledResourcesViewMode(): InstalledResourcesViewMode {
  return readViewMode(INSTALLED_RESOURCES_VIEW_STORAGE_KEY, 'list');
}

export function useInstalledResourcesViewMode() {
  return useStoredViewMode(INSTALLED_RESOURCES_VIEW_STORAGE_KEY, 'list');
}

export function useGameServerContentViewMode() {
  return useStoredViewMode(GAME_SERVER_CONTENT_VIEW_STORAGE_KEY, 'cards');
}

export function useGameServersListViewMode() {
  return useStoredViewMode(GAME_SERVERS_LIST_VIEW_STORAGE_KEY, 'cards');
}
