import type { LauncherInstance } from '@/api/client';

export function isServerManagedInstance(
  instance: Pick<LauncherInstance, 'managed_by_game_server_id' | 'content_locked'> | null | undefined,
): boolean {
  if (!instance) return false;
  return Boolean(instance.content_locked || instance.managed_by_game_server_id);
}
