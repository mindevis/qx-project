import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { ReactNode } from 'react';
import { api } from '@/api/client';
import { I18nProvider } from '@/i18n/I18nContext';
import { useModInstall } from './useModInstall';

const messageMocks = {
  success: vi.fn(),
  error: vi.fn(),
};

vi.mock('@/hooks/useMessage', () => ({
  useMessage: () => messageMocks,
}));

function wrapper({ children }: { children: ReactNode }) {
  return <I18nProvider>{children}</I18nProvider>;
}

const version = {
  id: 'ver-1',
  version_number: '1.0.0',
  game_versions: ['1.21'],
  loaders: ['forge'],
  files: [{ filename: 'mod.jar', url: 'https://example.com/mod.jar', primary: true }],
};

describe('useModInstall', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    messageMocks.success.mockReset();
    messageMocks.error.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('reports missing file', async () => {
    const { result } = renderHook(() => useModInstall('inst-1'), { wrapper });
    await act(async () => {
      const ok = await result.current.installVersion({
        source: 'modrinth',
        projectId: 'sodium',
        projectName: 'Sodium',
        version: { ...version, files: [] },
        resourceType: 'mod',
      });
      expect(ok).toBe(false);
    });
    expect(messageMocks.error).toHaveBeenCalled();
  });

  it('completes when install request succeeds', async () => {
    vi.spyOn(api, 'createModInstallRequest').mockResolvedValue({
      id: 'req-1',
      instance_id: 'inst-1',
      status: 'queued',
      created_at: 'now',
      updated_at: 'now',
    });
    vi.spyOn(api, 'getModInstallRequest')
      .mockResolvedValueOnce({
        id: 'req-1',
        instance_id: 'inst-1',
        status: 'dispatched',
        created_at: 'now',
        updated_at: 'now',
      })
      .mockResolvedValueOnce({
        id: 'req-1',
        instance_id: 'inst-1',
        status: 'completed',
        created_at: 'now',
        updated_at: 'now',
      });

    const { result } = renderHook(() => useModInstall('inst-1'), { wrapper });

    let ok = false;
    await act(async () => {
      const promise = result.current.installVersion({
        source: 'modrinth',
        projectId: 'sodium',
        projectName: 'Sodium',
        version,
        resourceType: 'mod',
      });
      await vi.advanceTimersByTimeAsync(1500);
      ok = await promise;
    });

    expect(ok).toBe(true);
    expect(messageMocks.success).toHaveBeenCalled();
    expect(result.current.installedVersionId).toBe('ver-1');
  });

  it('reports failure when install request fails', async () => {
    vi.spyOn(api, 'createModInstallRequest').mockResolvedValue({
      id: 'req-2',
      instance_id: 'inst-1',
      status: 'queued',
      created_at: 'now',
      updated_at: 'now',
    });
    vi.spyOn(api, 'getModInstallRequest').mockResolvedValue({
      id: 'req-2',
      instance_id: 'inst-1',
      status: 'failed',
      error_code: 'INSTALL_FAILED',
      created_at: 'now',
      updated_at: 'now',
    });

    const { result } = renderHook(() => useModInstall('inst-1'), { wrapper });

    let ok = true;
    await act(async () => {
      const promise = result.current.installVersion({
        source: 'modrinth',
        projectId: 'sodium',
        projectName: 'Sodium',
        version,
        resourceType: 'mod',
      });
      await vi.advanceTimersByTimeAsync(1500);
      ok = await promise;
    });

    expect(ok).toBe(false);
    expect(messageMocks.error).toHaveBeenCalled();
  });

  it('reports device required when create fails with forbidden', async () => {
    vi.spyOn(api, 'createModInstallRequest').mockRejectedValue(new Error('device not linked'));

    const { result } = renderHook(() => useModInstall('inst-1'), { wrapper });

    await act(async () => {
      const ok = await result.current.installVersion({
        source: 'modrinth',
        projectId: 'sodium',
        projectName: 'Sodium',
        version,
        resourceType: 'mod',
      });
      expect(ok).toBe(false);
    });

    expect(messageMocks.error).toHaveBeenCalled();
  });

  it('resetInstalled clears installed version id', async () => {
    vi.spyOn(api, 'createModInstallRequest').mockResolvedValue({
      id: 'req-3',
      instance_id: 'inst-1',
      status: 'queued',
      created_at: 'now',
      updated_at: 'now',
    });
    vi.spyOn(api, 'getModInstallRequest').mockResolvedValue({
      id: 'req-3',
      instance_id: 'inst-1',
      status: 'completed',
      created_at: 'now',
      updated_at: 'now',
    });

    const { result } = renderHook(() => useModInstall('inst-1'), { wrapper });

    await act(async () => {
      const promise = result.current.installVersion({
        source: 'modrinth',
        projectId: 'sodium',
        projectName: 'Sodium',
        version,
        resourceType: 'mod',
      });
      await vi.runAllTimersAsync();
      await promise;
    });

    expect(result.current.installedVersionId).toBe('ver-1');

    act(() => {
      result.current.resetInstalled();
    });

    expect(result.current.installedVersionId).toBeUndefined();
  });
});
