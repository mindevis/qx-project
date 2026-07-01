import { describe, expect, it } from 'vitest';
import {
  LAUNCH_ACTIVE_STATUSES,
  LAUNCH_TERMINAL_STATUSES,
  getLaunchErrorKey,
  isLaunchTerminal,
} from './launchProgress';

describe('launchProgress', () => {
  it('identifies terminal launch statuses', () => {
    expect(isLaunchTerminal('completed')).toBe(true);
    expect(isLaunchTerminal('failed')).toBe(true);
    expect(isLaunchTerminal('expired')).toBe(true);
    expect(isLaunchTerminal('running')).toBe(false);
    expect(isLaunchTerminal('preparing')).toBe(false);
  });

  it('tracks active statuses', () => {
    expect(LAUNCH_TERMINAL_STATUSES.has('failed')).toBe(true);
    expect(LAUNCH_ACTIVE_STATUSES.has('downloading')).toBe(true);
    expect(LAUNCH_ACTIVE_STATUSES.has('completed')).toBe(false);
  });

  it('maps launch error codes to i18n keys', () => {
    expect(getLaunchErrorKey('MOJANG_SESSION')).toBe('launcher.launchErrorCodes.MOJANG_SESSION');
    expect(getLaunchErrorKey(undefined)).toBeUndefined();
  });
});
