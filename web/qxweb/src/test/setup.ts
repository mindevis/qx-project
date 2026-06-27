import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';
import { cleanup, configure } from '@testing-library/react';
import { message } from 'antd';

configure({ asyncUtilTimeout: 5000 });

vi.mock('@/hooks/useMessage', () => ({
  useMessage: () => message,
}));

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

class ResizeObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

vi.stubGlobal('ResizeObserver', ResizeObserverMock);

afterEach(() => {
  cleanup();
  localStorage.clear();
  window.history.pushState({}, '', '/');
  mockMatchMedia();
  message.destroy();
  // Re-apply after tests that call vi.unstubAllGlobals() (only unstubs fetch).
  vi.stubGlobal('ResizeObserver', ResizeObserverMock);
});
