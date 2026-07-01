import { describe, expect, it } from 'vitest';
import { formatDownloadCount, formatFileSize } from '@/lib/formatFileSize';

describe('formatFileSize', () => {
  it('formats bytes and megabytes', () => {
    expect(formatFileSize(512)).toBe('512 B');
    expect(formatFileSize(2 * 1024 * 1024)).toBe('2.0 MB');
  });

  it('returns empty for missing values', () => {
    expect(formatFileSize()).toBe('');
    expect(formatFileSize(0)).toBe('');
  });
});

describe('formatDownloadCount', () => {
  it('abbreviates large counts', () => {
    expect(formatDownloadCount(1500)).toBe('1.5K');
    expect(formatDownloadCount(2_500_000)).toBe('2.5M');
  });
});
