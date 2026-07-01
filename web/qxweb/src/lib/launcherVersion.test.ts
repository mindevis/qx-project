import { describe, expect, it } from 'vitest';
import { compareVersions, isUpdateAvailable, normalizeVersion } from './launcherVersion';

describe('launcherVersion', () => {
  it('normalizes leading v prefix', () => {
    expect(normalizeVersion('v1.2.3')).toBe('1.2.3');
  });

  it('compares semantic versions', () => {
    expect(compareVersions('0.1.0', '0.2.0')).toBeLessThan(0);
    expect(compareVersions('1.0.0', '0.9.9')).toBeGreaterThan(0);
    expect(compareVersions('v1.2.3', '1.2.3')).toBe(0);
  });

  it('detects when an update is available', () => {
    expect(isUpdateAvailable('0.1.0', '0.2.0')).toBe(true);
    expect(isUpdateAvailable('1.0.0', '1.0.0')).toBe(false);
    expect(isUpdateAvailable(undefined, '1.0.0')).toBe(true);
  });
});
