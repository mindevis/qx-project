import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { checkBackendHealth } from '@/api/client';

export const BACKEND_HEALTH_POLL_MS = 10_000;

function scheduleDeferred(task: () => void): () => void {
  if (import.meta.env.MODE === 'test') {
    task();
    return () => {};
  }
  if (typeof window.requestIdleCallback === 'function') {
    const id = window.requestIdleCallback(() => task(), { timeout: 2_000 });
    return () => window.cancelIdleCallback(id);
  }
  const id = window.setTimeout(task, 1);
  return () => window.clearTimeout(id);
}

type BackendStatusState = {
  available: boolean;
};

const BackendStatusContext = createContext<BackendStatusState | null>(null);

export function BackendStatusProvider({
  children,
  pollIntervalMs = BACKEND_HEALTH_POLL_MS,
}: {
  children: ReactNode;
  pollIntervalMs?: number;
}) {
  const [available, setAvailable] = useState(true);

  const refresh = useCallback(async () => {
    setAvailable(await checkBackendHealth());
  }, []);

  useEffect(() => {
    let cancelled = false;
    const run = async () => {
      if (!cancelled) {
        await refresh();
      }
    };

    const cancelSchedule = scheduleDeferred(run);

    const id = window.setInterval(() => void refresh(), pollIntervalMs);
    return () => {
      cancelled = true;
      cancelSchedule();
      window.clearInterval(id);
    };
  }, [pollIntervalMs, refresh]);

  const value = useMemo(() => ({ available }), [available]);

  return (
    <BackendStatusContext.Provider value={value}>{children}</BackendStatusContext.Provider>
  );
}

export function useBackendStatus() {
  const ctx = useContext(BackendStatusContext);
  if (!ctx) {
    throw new Error('useBackendStatus must be used within BackendStatusProvider');
  }
  return ctx;
}
