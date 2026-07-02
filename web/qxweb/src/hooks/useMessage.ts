import type { MessageInstance, MessageType } from 'antd/es/message/interface';

function createNoopMessageType(): MessageType {
  const destroy = (() => {}) as MessageType;
  destroy.then = (onFulfilled, onRejected) =>
    Promise.resolve(false).then(onFulfilled, onRejected);
  return destroy;
}

const noop: MessageInstance['success'] = () => createNoopMessageType();

/** No-op message API — top toasts are disabled app-wide. */
const disabledMessage: MessageInstance = {
  success: noop,
  error: noop,
  info: noop,
  warning: noop,
  loading: noop,
  open: () => createNoopMessageType(),
  destroy: () => {},
};

/** Toast API used across the app (currently disabled — calls are no-ops). */
export function useMessage(): MessageInstance {
  return disabledMessage;
}
