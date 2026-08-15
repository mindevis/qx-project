import { describe, expect, it } from 'vitest';
import { formatCompactCount } from './formatCompactCount';

describe('formatCompactCount', () => {
  it('keeps small numbers as-is', () => {
    expect(formatCompactCount(0)).toBe('0');
    expect(formatCompactCount(999)).toBe('999');
  });

  it('formats thousands and millions compactly', () => {
    expect(formatCompactCount(1200)).toBe('1.2K');
    expect(formatCompactCount(15_400)).toBe('15K');
    expect(formatCompactCount(1_200_000)).toBe('1.2M');
    expect(formatCompactCount(12_000_000)).toBe('12M');
  });
});
