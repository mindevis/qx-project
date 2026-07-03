type CacheEntry<T> = {
  value: T;
  expiresAt: number;
};

export type TtlCache<T> = {
  get: (key: string) => T | undefined;
  set: (key: string, value: T) => void;
  getOrLoad: (key: string, loader: () => Promise<T>) => Promise<T>;
  clear: () => void;
  delete: (key: string) => void;
};

export function createTtlCache<T>(ttlMs: number, maxEntries = 500): TtlCache<T> {
  const store = new Map<string, CacheEntry<T>>();
  const inflight = new Map<string, Promise<T>>();

  const isValid = (entry: CacheEntry<T>) => Date.now() < entry.expiresAt;

  const evictIfNeeded = () => {
    while (store.size > maxEntries) {
      const first = store.keys().next().value;
      if (first === undefined) break;
      store.delete(first);
    }
  };

  return {
    get(key: string) {
      const entry = store.get(key);
      if (!entry) return undefined;
      if (!isValid(entry)) {
        store.delete(key);
        return undefined;
      }
      return entry.value;
    },

    set(key: string, value: T) {
      evictIfNeeded();
      store.set(key, { value, expiresAt: Date.now() + ttlMs });
    },

    async getOrLoad(key: string, loader: () => Promise<T>) {
      const cached = this.get(key);
      if (cached !== undefined) return cached;

      const pending = inflight.get(key);
      if (pending) return pending;

      const promise = loader()
        .then((value) => {
          this.set(key, value);
          return value;
        })
        .finally(() => {
          inflight.delete(key);
        });
      inflight.set(key, promise);
      return promise;
    },

    clear() {
      store.clear();
      inflight.clear();
    },

    delete(key: string) {
      store.delete(key);
      inflight.delete(key);
    },
  };
}
