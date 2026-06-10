export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const LEVEL_RANK: Record<LogLevel, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

function normalizeLevel(raw: string | undefined): LogLevel {
  const value = (raw ?? 'info').toLowerCase();
  if (value === 'warning') return 'warn';
  if (value in LEVEL_RANK) return value as LogLevel;
  return 'info';
}

const env = {
  logLevel: () => import.meta.env.VITE_LOG_LEVEL,
  isDev: () => import.meta.env.DEV,
};

function resolveMinLevel(): LogLevel {
  const fromEnv = env.logLevel();
  if (fromEnv) return normalizeLevel(fromEnv);
  return env.isDev() ? 'debug' : 'info';
}

const config = {
  minLevel: resolveMinLevel,
};

function shouldLog(level: LogLevel): boolean {
  return LEVEL_RANK[level] >= LEVEL_RANK[config.minLevel()];
}

function write(level: LogLevel, message: string, details?: unknown) {
  if (!shouldLog(level)) return;
  const prefix = `[QX][${level.toUpperCase()}]`;
  const args = details === undefined ? [prefix, message] : [prefix, message, details];
  switch (level) {
    case 'debug':
      console.debug(...args);
      break;
    case 'info':
      console.info(...args);
      break;
    case 'warn':
      console.warn(...args);
      break;
    case 'error':
      console.error(...args);
      break;
  }
}

export const logger = {
  debug: (message: string, details?: unknown) => write('debug', message, details),
  info: (message: string, details?: unknown) => write('info', message, details),
  warn: (message: string, details?: unknown) => write('warn', message, details),
  error: (message: string, details?: unknown) => write('error', message, details),
};

/** @internal exported for unit tests */
export const __test__ = {
  normalizeLevel,
  shouldLog,
  resolveMinLevel,
  setMinLevel(level: LogLevel) {
    config.minLevel = () => level;
  },
  reset() {
    config.minLevel = resolveMinLevel;
    env.logLevel = () => import.meta.env.VITE_LOG_LEVEL;
    env.isDev = () => import.meta.env.DEV;
  },
  setEnv(logLevel?: string, isDev = true) {
    env.logLevel = () => logLevel;
    env.isDev = () => isDev;
  },
};
