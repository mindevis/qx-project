import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { __test__, logger } from './logger';

describe('logger', () => {
  beforeEach(() => {
    vi.spyOn(console, 'debug').mockImplementation(() => {});
    vi.spyOn(console, 'info').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    __test__.reset();
    vi.restoreAllMocks();
  });

  it('normalizes warning alias', () => {
    expect(__test__.normalizeLevel('WARNING')).toBe('warn');
    expect(__test__.normalizeLevel('unknown')).toBe('info');
  });

  it('logs at each level when threshold allows', () => {
    __test__.setMinLevel('debug');

    logger.debug('dbg', { a: 1 });
    logger.info('inf');
    logger.warn('wrn');
    logger.error('err');

    expect(console.debug).toHaveBeenCalled();
    expect(console.info).toHaveBeenCalled();
    expect(console.warn).toHaveBeenCalled();
    expect(console.error).toHaveBeenCalled();
  });

  it('skips output below threshold', () => {
    __test__.setMinLevel('error');

    logger.info('hidden');

    expect(console.info).not.toHaveBeenCalled();
  });

  it('shouldLog respects min level', () => {
    __test__.setMinLevel('warn');
    expect(__test__.shouldLog('info')).toBe(false);
    expect(__test__.shouldLog('error')).toBe(true);
  });

  it('logs message without details', () => {
    __test__.setMinLevel('info');
    logger.info('plain');
    expect(console.info).toHaveBeenCalledWith('[QX][INFO]', 'plain');
  });

  it('normalizes all known levels', () => {
    expect(__test__.normalizeLevel(undefined)).toBe('info');
    expect(__test__.normalizeLevel('debug')).toBe('debug');
    expect(__test__.normalizeLevel('info')).toBe('info');
    expect(__test__.normalizeLevel('warn')).toBe('warn');
    expect(__test__.normalizeLevel('error')).toBe('error');
  });

  it('resolveMinLevel uses env and production defaults', () => {
    __test__.setEnv('error');
    expect(__test__.resolveMinLevel()).toBe('error');

    __test__.setEnv(undefined, false);
    expect(__test__.resolveMinLevel()).toBe('info');

    __test__.setEnv(undefined, true);
    expect(__test__.resolveMinLevel()).toBe('debug');
  });

  it('reset restores default env readers', () => {
    __test__.setEnv('error', false);
    __test__.reset();
    expect(__test__.resolveMinLevel()).toBe('debug');
  });
});
