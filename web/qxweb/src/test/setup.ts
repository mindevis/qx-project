import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeEach, vi } from 'vitest';

function installLocalStoragePolyfill() {
  const target = typeof window === 'undefined' ? globalThis : window;
  try {
    if (target.localStorage?.getItem && target.localStorage.removeItem) {
      return;
    }
  } catch {
    // jsdom/Node 24 can expose a broken localStorage getter.
  }
  const store = new Map<string, string>();
  const memory: Storage = {
    getItem: (key) => store.get(key) ?? null,
    setItem: (key, value) => {
      store.set(key, String(value));
    },
    removeItem: (key) => {
      store.delete(key);
    },
    clear: () => store.clear(),
    key: (index) => [...store.keys()][index] ?? null,
    get length() {
      return store.size;
    },
  };
  Object.defineProperty(target, 'localStorage', {
    configurable: true,
    writable: true,
    value: memory,
  });
}

installLocalStoragePolyfill();
import { act, cleanup, configure } from '@testing-library/react';
import { message, Modal } from 'antd';
import { clearTokens } from '@/api/client';
import { __test__ as loggerTest } from '@/lib/logger';
import { installCanvasMocks } from '@/test/canvas-mock';
import { installNavigationMock } from '@/test/navigation-mock';
import { resetTestMessage } from '@/test/test-message';
import { clearMcVersionsCache } from '@/lib/mcVersionsCache';
import { clearModCatalogCaches } from '@/lib/modCatalogCache';
import { clearGameServerSyncTargetsCache } from '@/lib/gameServerSyncTargets';
import { clearGameServerVersionsCache } from '@/lib/gameServerVersions';

configure({ asyncUtilTimeout: 5000 });

beforeEach(() => {
  loggerTest.setMinLevel('error');
  installNavigationMock();
});

vi.mock('skinview3d', async () => {
  const { skinview3dMock } = await import('@/test/skinview3d-mock');
  return skinview3dMock;
});

vi.mock('@/hooks/useMessage', async () => {
  const { testMessage } = await import('@/test/test-message');
  return { useMessage: () => testMessage };
});

vi.mock('@/hooks/useModal', async () => {
  const { Modal } = await import('antd');
  return {
    useModal: () => ({
      confirm: (config: Parameters<typeof Modal.confirm>[0]) => Modal.confirm(config),
      info: (config: Parameters<typeof Modal.info>[0]) => Modal.info(config),
      success: (config: Parameters<typeof Modal.success>[0]) => Modal.success(config),
      error: (config: Parameters<typeof Modal.error>[0]) => Modal.error(config),
      warning: (config: Parameters<typeof Modal.warning>[0]) => Modal.warning(config),
    }),
  };
});

vi.mock('@/backend/BackendStatusContext', () => ({
  BackendStatusProvider: ({ children }: { children: React.ReactNode }) => children,
  useBackendStatus: vi.fn(() => ({ available: true })),
  BACKEND_HEALTH_POLL_MS: 10_000,
}));

function mockMatchMedia() {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

mockMatchMedia();

// jsdom does not implement getComputedStyle(..., '::before'|'::after'); antd/rc-* call it often.
const nativeGetComputedStyle = window.getComputedStyle.bind(window);
window.getComputedStyle = (element, _pseudoElement?) => nativeGetComputedStyle(element);

installCanvasMocks();

class ResizeObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

vi.stubGlobal('ResizeObserver', ResizeObserverMock);

async function flushReactWork() {
  await act(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    await new Promise<void>((resolve) => setImmediate(resolve));
  });
}

afterEach(async () => {
  cleanup();
  Modal.destroyAll();
  clearTokens();
  localStorage.clear();
  window.history.pushState({}, '', '/');
  mockMatchMedia();
  message.destroy();
  resetTestMessage();
  clearMcVersionsCache();
  clearModCatalogCaches();
  clearGameServerSyncTargetsCache();
  clearGameServerVersionsCache();
  vi.stubGlobal('ResizeObserver', ResizeObserverMock);
  await flushReactWork();
});

afterAll(async () => {
  loggerTest.reset();
  await flushReactWork();
});
