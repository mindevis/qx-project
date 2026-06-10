import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { ConfigProvider, theme } from 'antd';
import ruRU from 'antd/locale/ru_RU';

export type ThemeMode = 'light' | 'dark';

const STORAGE_KEY = 'qxweb-theme';

type ThemeContextValue = {
  mode: ThemeMode;
  isDark: boolean;
  setMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

function readStoredMode(): ThemeMode {
  /* v8 ignore next 3 */
  if (typeof window === 'undefined') {
    return 'light';
  }

  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === 'light' || stored === 'dark') {
    return stored;
  }

  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(readStoredMode);

  const setMode = useCallback((nextMode: ThemeMode) => {
    setModeState(nextMode);
    window.localStorage.setItem(STORAGE_KEY, nextMode);
  }, []);

  const toggleTheme = useCallback(() => {
    setMode(mode === 'dark' ? 'light' : 'dark');
  }, [mode, setMode]);

  useEffect(() => {
    document.documentElement.dataset.theme = mode;
  }, [mode]);

  const value = useMemo(
    () => ({
      mode,
      isDark: mode === 'dark',
      setMode,
      toggleTheme,
    }),
    [mode, setMode, toggleTheme],
  );

  const antTheme = useMemo(
    () => ({
      algorithm: mode === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
    }),
    [mode],
  );

  return (
    <ThemeContext.Provider value={value}>
      <ConfigProvider locale={ruRU} theme={antTheme}>
        {children}
      </ConfigProvider>
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
}
