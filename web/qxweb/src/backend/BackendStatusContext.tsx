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
    void refresh();
    const id = window.setInterval(() => void refresh(), pollIntervalMs);
    return () => window.clearInterval(id);
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
