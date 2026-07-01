import { ApiRequestError, isBackendUnavailableError } from '@/api/client';

type ModCatalogErrorT = (key: string) => string;

/** Maps mod catalog API errors to user-facing messages (never generic backend-down for CurseForge). */
export function formatModCatalogError(
  error: unknown,
  t: ModCatalogErrorT,
  fallbackKey: 'qxmods.browseFailed' | 'qxmods.searchFailed' | 'qxmods.versionsFailed',
): string {
  if (isBackendUnavailableError(error)) {
    return t('backend.title');
  }
  if (error instanceof ApiRequestError) {
    if (error.apiCode === 'SOURCE_UNAVAILABLE') {
      return t('qxmods.curseforgeDisabled');
    }
    if (error.apiCode === 'UPSTREAM_UNAVAILABLE' || error.apiCode === 'CURSEFORGE_UNAVAILABLE') {
      if (error.message.toLowerCase().includes('curseforge')) {
        return error.message;
      }
      return error.message || t('qxmods.upstreamFailed');
    }
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return t(fallbackKey);
}
