import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { createTtlCache } from './ttlCache';

describe('createTtlCache', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns cached value before ttl expires', async () => {
    const cache = createTtlCache<string>(1000);
    const loader = vi.fn().mockResolvedValue('value');

    await expect(cache.getOrLoad('key', loader)).resolves.toBe('value');
    await expect(cache.getOrLoad('key', loader)).resolves.toBe('value');
    expect(loader).toHaveBeenCalledTimes(1);
  });

  it('reloads after ttl expires', async () => {
    const cache = createTtlCache<string>(1000);
    const loader = vi.fn().mockResolvedValueOnce('first').mockResolvedValueOnce('second');

    await cache.getOrLoad('key', loader);
    vi.advanceTimersByTime(1001);
    await expect(cache.getOrLoad('key', loader)).resolves.toBe('second');
    expect(loader).toHaveBeenCalledTimes(2);
  });

  it('deduplicates concurrent loads', async () => {
    const cache = createTtlCache<string>(1000);
    let resolve!: (value: string) => void;
    const loader = vi.fn(
      () =>
        new Promise<string>((resolvePromise) => {
          resolve = resolvePromise;
        }),
    );

    const first = cache.getOrLoad('key', loader);
    const second = cache.getOrLoad('key', loader);
    resolve('done');

    await expect(Promise.all([first, second])).resolves.toEqual(['done', 'done']);
    expect(loader).toHaveBeenCalledTimes(1);
  });
});
