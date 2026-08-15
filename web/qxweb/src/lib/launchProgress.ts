export const LAUNCH_TERMINAL_STATUSES = new Set(['completed', 'failed', 'expired']);

export const LAUNCH_ACTIVE_STATUSES = new Set([
  'queued',
  'dispatched',
  'preparing',
  'downloading',
  'launching',
  'running',
]);

export type LaunchAccountMode = 'offline' | 'licensed';

export type LaunchProgressState = {
  instanceId: string;
  requestId: string;
  status: string;
  accountMode: LaunchAccountMode;
  errorCode?: string;
  needsMojangRelink?: boolean;
};

export function isLaunchTerminal(status: string): boolean {
  return LAUNCH_TERMINAL_STATUSES.has(status);
}

export function isLaunchStarted(status: string): boolean {
  return status === 'running' || isLaunchTerminal(status);
}

export function getLaunchErrorKey(errorCode?: string): string | undefined {
  if (!errorCode) {
    return undefined;
  }
  const key = `launcher.launchErrorCodes.${errorCode}`;
  return key;
}
