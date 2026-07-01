import { vi } from 'vitest';
import type { MessageInstance } from 'antd/es/message/interface';

function createMessageMethod() {
  const fn = vi.fn(() => vi.fn());
  return fn as MessageInstance['success'];
}

/** Context-free message API for Vitest (avoids antd static `message` warnings). */
export const testMessage: MessageInstance = {
  success: createMessageMethod(),
  error: createMessageMethod(),
  info: createMessageMethod(),
  warning: createMessageMethod(),
  loading: createMessageMethod(),
  open: vi.fn(),
  destroy: vi.fn(),
};

export function resetTestMessage() {
  for (const method of Object.values(testMessage)) {
    if (typeof method === 'function' && 'mockClear' in method) {
      (method as ReturnType<typeof vi.fn>).mockClear();
    }
  }
}
