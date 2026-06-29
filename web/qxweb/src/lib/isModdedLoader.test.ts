import { describe, expect, it } from 'vitest';
import { isModdedLauncherLoader } from './isModdedLoader';

describe('isModdedLauncherLoader', () => {
  it('returns true for modded loaders', () => {
    expect(isModdedLauncherLoader('forge')).toBe(true);
    expect(isModdedLauncherLoader('neoforge')).toBe(true);
    expect(isModdedLauncherLoader('fabric')).toBe(true);
    expect(isModdedLauncherLoader('quilt')).toBe(true);
  });

  it('returns false for vanilla and unknown', () => {
    expect(isModdedLauncherLoader('vanilla')).toBe(false);
    expect(isModdedLauncherLoader('paper')).toBe(false);
    expect(isModdedLauncherLoader('')).toBe(false);
  });
});
