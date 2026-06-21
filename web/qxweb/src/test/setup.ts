import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';
import { cleanup, configure } from '@testing-library/react';
import { message } from 'antd';

configure({ asyncUtilTimeout: 5000 });

vi.mock('@/hooks/useMessage', () => ({
  useMessage: () => message,
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

afterEach(() => {
  cleanup();
  localStorage.clear();
  window.history.pushState({}, '', '/');
  mockMatchMedia();
});
