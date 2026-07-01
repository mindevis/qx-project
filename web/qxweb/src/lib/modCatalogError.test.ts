import { describe, expect, it } from 'vitest';
import { ApiRequestError, API_ERROR_BACKEND_UNAVAILABLE } from '@/api/client';
import { formatModCatalogError } from './modCatalogError';

const t = (key: string) => key;

describe('formatModCatalogError', () => {
  it('maps SOURCE_UNAVAILABLE to curseforge disabled message', () => {
    const err = new ApiRequestError('curseforge api key not configured', undefined, 'SOURCE_UNAVAILABLE');
    expect(formatModCatalogError(err, t, 'qxmods.browseFailed')).toBe('qxmods.curseforgeDisabled');
  });

  it('passes through CurseForge upstream API details', () => {
    const err = new ApiRequestError('curseforge: status 403: forbidden', undefined, 'CURSEFORGE_UNAVAILABLE');
    expect(formatModCatalogError(err, t, 'qxmods.browseFailed')).toBe('curseforge: status 403: forbidden');
  });

  it('uses backend title only for true backend-down errors', () => {
    const err = new ApiRequestError('Backend unavailable', API_ERROR_BACKEND_UNAVAILABLE);
    expect(formatModCatalogError(err, t, 'qxmods.browseFailed')).toBe('backend.title');
  });
});
