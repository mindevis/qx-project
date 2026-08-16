import { describe, expect, it, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import {
  GAME_SERVER_CONTENT_VIEW_STORAGE_KEY,
  INSTALLED_RESOURCES_VIEW_STORAGE_KEY,
  readInstalledResourcesViewMode,
  useGameServerContentViewMode,
  useInstalledResourcesViewMode,
} from './installedResourcesView';

describe('installedResourcesView', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it('defaults to list when storage is empty', () => {
    expect(readInstalledResourcesViewMode()).toBe('list');
  });

  it('reads stored view mode', () => {
    window.localStorage.setItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY, 'cards');
    expect(readInstalledResourcesViewMode()).toBe('cards');
  });

  it('falls back to list for invalid stored value', () => {
    window.localStorage.setItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY, 'grid');
    expect(readInstalledResourcesViewMode()).toBe('list');
  });

  it('persists view mode changes', () => {
    const { result } = renderHook(() => useInstalledResourcesViewMode());

    expect(result.current.viewMode).toBe('list');

    act(() => {
      result.current.setViewMode('cards');
    });

    expect(result.current.viewMode).toBe('cards');
    expect(window.localStorage.getItem(INSTALLED_RESOURCES_VIEW_STORAGE_KEY)).toBe('cards');
  });

  it('defaults game server content view to cards and persists table mode', () => {
    const { result } = renderHook(() => useGameServerContentViewMode());

    expect(result.current.viewMode).toBe('cards');

    act(() => {
      result.current.setViewMode('list');
    });

    expect(result.current.viewMode).toBe('list');
    expect(window.localStorage.getItem(GAME_SERVER_CONTENT_VIEW_STORAGE_KEY)).toBe('list');
  });
});
