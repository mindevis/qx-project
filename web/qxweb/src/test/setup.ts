import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, vi } from 'vitest';
import { act, cleanup, configure } from '@testing-library/react';
import { message } from 'antd';
import { clearTokens } from '@/api/client';

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

async function flushReactWork() {
  await act(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    await new Promise<void>((resolve) => setImmediate(resolve));
  });
}

afterEach(async () => {
  cleanup();
  clearTokens();
  localStorage.clear();
  window.history.pushState({}, '', '/');
  mockMatchMedia();
  message.destroy();
  vi.stubGlobal('ResizeObserver', ResizeObserverMock);
  await flushReactWork();
});

afterAll(async () => {
  await flushReactWork();
});
