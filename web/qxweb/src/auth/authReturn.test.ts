import { describe, expect, it } from 'vitest';
import { buildAuthReturnPath, isSafeReturnPath } from './authReturn';

describe('authReturn', () => {
  it('accepts in-app paths', () => {
    expect(isSafeReturnPath('/launcher')).toBe(true);
    expect(isSafeReturnPath('/launcher/link?device=abc')).toBe(true);
    expect(buildAuthReturnPath('/servers', '?page=2', '#top')).toBe('/servers?page=2#top');
  });

  it('rejects auth routes and external paths', () => {
    expect(isSafeReturnPath('/auth/login')).toBe(false);
    expect(isSafeReturnPath('//evil.com')).toBe(false);
    expect(buildAuthReturnPath('/auth/register')).toBe('/');
  });
});
