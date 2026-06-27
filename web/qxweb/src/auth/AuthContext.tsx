import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import {
  api,
  clearLinkedDevice,
  clearTokens,
  loadTokens,
  saveTokens,
  type TokenResponse,
  type UserProfile,
} from '@/api/client';
import { logger } from '@/lib/logger';

type AuthState = {
  user: UserProfile | null;
  loading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshProfile: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshProfile = useCallback(async () => {
    const tokens = loadTokens();
    if (!tokens) {
      setUser(null);
      return;
    }
    const profile = await api.me();
    setUser(profile);
  }, []);

  useEffect(() => {
    (async () => {
      try {
        if (loadTokens()) {
          logger.debug('restoring session');
          await refreshProfile();
        }
      } catch {
        logger.warn('session restore failed');
        clearTokens();
        setUser(null);
      } finally {
        setLoading(false);
      }
    })();
  }, [refreshProfile]);

  const applyTokens = useCallback(
    async (tokens: TokenResponse) => {
      saveTokens(tokens);
      await refreshProfile();
    },
    [refreshProfile],
  );

  const login = useCallback(
    async (email: string, password: string) => {
      const tokens = await api.login({ email, password });
      await applyTokens(tokens);
      logger.info('user logged in', { email });
    },
    [applyTokens],
  );

  const register = useCallback(
    async (email: string, password: string) => {
      const tokens = await api.register({ email, password });
      await applyTokens(tokens);
    },
    [applyTokens],
  );

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      logger.debug('logout request failed');
    } finally {
      clearTokens();
      clearLinkedDevice();
      setUser(null);
      logger.info('user logged out');
    }
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      user,
      loading,
      isAuthenticated: !!user,
      login,
      register,
      logout,
      refreshProfile,
    }),
    [user, loading, login, register, logout, refreshProfile],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return ctx;
}
