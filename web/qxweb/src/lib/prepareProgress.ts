export const PREPARE_TERMINAL_STATUSES = new Set(['completed', 'failed', 'expired']);

export const PREPARE_ACTIVE_STATUSES = new Set(['queued', 'preparing', 'downloading']);

export type PrepareProgressState = {
  instanceId: string;
  requestId: string;
  status: string;
  errorCode?: string;
  progressMessage?: string;
};

export function isPrepareTerminal(status: string): boolean {
  return PREPARE_TERMINAL_STATUSES.has(status);
}

export function isPrepareActive(status: string): boolean {
  return PREPARE_ACTIVE_STATUSES.has(status);
}
