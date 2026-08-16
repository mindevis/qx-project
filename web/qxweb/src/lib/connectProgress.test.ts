import { describe, expect, it } from 'vitest';
import { CONNECT_STEPS, connectFileProgressKey } from './connectProgress';

describe('connectProgress', () => {
  it('keeps connect steps in order', () => {
    expect(CONNECT_STEPS).toEqual(['creating', 'preparing', 'clientMods', 'syncing', 'launching']);
  });

  it('maps launcher file phases to i18n keys', () => {
    expect(connectFileProgressKey('java runtime')).toBe('monitoring.connectProgress.files.java');
    expect(connectFileProgressKey('client jar')).toBe('monitoring.connectProgress.files.client');
    expect(connectFileProgressKey('libraries')).toBe('monitoring.connectProgress.files.libraries');
    expect(connectFileProgressKey('natives')).toBe('monitoring.connectProgress.files.natives');
    expect(connectFileProgressKey('assets')).toBe('monitoring.connectProgress.files.assets');
    expect(connectFileProgressKey('forge installer')).toBe('monitoring.connectProgress.files.loader');
    expect(connectFileProgressKey('')).toBeUndefined();
    expect(connectFileProgressKey(undefined)).toBeUndefined();
  });
});
