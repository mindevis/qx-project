export type ConnectProgressStep =
  | 'creating'
  | 'preparing'
  | 'clientMods'
  | 'syncing'
  | 'launching';

export const CONNECT_STEPS: ConnectProgressStep[] = [
  'creating',
  'preparing',
  'clientMods',
  'syncing',
  'launching',
];

export function connectFileProgressKey(message?: string): string | undefined {
  const raw = (message ?? '').trim().toLowerCase();
  if (!raw) return undefined;
  if (raw === 'java runtime' || raw.includes('java')) {
    return 'monitoring.connectProgress.files.java';
  }
  if (raw === 'client jar' || raw.includes('client')) {
    return 'monitoring.connectProgress.files.client';
  }
  if (raw === 'libraries' || raw.includes('librar')) {
    return 'monitoring.connectProgress.files.libraries';
  }
  if (raw === 'natives' || raw.includes('native')) {
    return 'monitoring.connectProgress.files.natives';
  }
  if (raw === 'assets' || raw.includes('asset')) {
    return 'monitoring.connectProgress.files.assets';
  }
  if (raw.includes('installer') || raw.includes('loader') || raw.includes('forge') || raw.includes('fabric')) {
    return 'monitoring.connectProgress.files.loader';
  }
  return undefined;
}
