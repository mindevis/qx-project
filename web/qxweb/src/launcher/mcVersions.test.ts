import { describe, expect, it } from 'vitest';
import { DEFAULT_MC_VERSION, MVP_MC_VERSIONS } from './mcVersions';

describe('mcVersions', () => {
  it('lists MVP releases', () => {
    expect(MVP_MC_VERSIONS).toContain('1.20.4');
    expect(MVP_MC_VERSIONS).toContain('1.21');
    expect(DEFAULT_MC_VERSION).toBe('1.21');
  });
});
