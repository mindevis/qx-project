import { describe, expect, it } from 'vitest';
import { officialAccountBodyUrl, officialAccountSkinUrl } from './mojangSkin';

describe('officialAccountBodyUrl', () => {
  it('prefers uuid over username', () => {
    expect(officialAccountBodyUrl('uuid-notchy', 'qDevis')).toBe(
      'https://mc-heads.net/body/uuid-notchy',
    );
  });

  it('falls back to username', () => {
    expect(officialAccountBodyUrl(undefined, 'qDevis')).toBe('https://mc-heads.net/body/qDevis');
  });

  it('returns undefined when both are empty', () => {
    expect(officialAccountBodyUrl('  ', '')).toBeUndefined();
    expect(officialAccountBodyUrl()).toBeUndefined();
  });
});

describe('officialAccountSkinUrl', () => {
  it('builds a CORS-friendly PNG url for the 3D viewer', () => {
    expect(officialAccountSkinUrl('abc-def', 'Steve')).toBe('https://mc-heads.net/skin/abc-def');
    expect(officialAccountSkinUrl(undefined, 'Steve')).toBe('https://mc-heads.net/skin/Steve');
    expect(officialAccountSkinUrl('  ')).toBeUndefined();
  });
});
