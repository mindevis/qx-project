import { describe, expect, it } from 'vitest';
import {
  DEFAULT_MC_VERSION,
  FALLBACK_MC_VERSIONS,
  groupMcVersionOptions,
  pickDefaultMcVersion,
} from './mcVersions';

describe('mcVersions', () => {
  it('exposes fallback releases', () => {
    expect(FALLBACK_MC_VERSIONS).toContain('1.20.4');
    expect(FALLBACK_MC_VERSIONS).toContain('1.21');
    expect(DEFAULT_MC_VERSION).toBe('1.21');
  });

  it('picks latest release from manifest', () => {
    expect(
      pickDefaultMcVersion(
        { release: '1.21.4', snapshot: '25w02a' },
        [{ id: '1.21.4', type: 'release' }],
      ),
    ).toBe('1.21.4');
  });

  it('groups versions by type', () => {
    const groups = groupMcVersionOptions(
      [
        { id: '1.21.4', type: 'release' },
        { id: '25w02a', type: 'snapshot' },
        { id: 'b1.8', type: 'old_beta' },
      ],
      (type) => type,
    );
    expect(groups).toHaveLength(3);
    expect(groups[0]?.options[0]?.value).toBe('1.21.4');
    expect(groups[1]?.options[0]?.value).toBe('25w02a');
  });
});
