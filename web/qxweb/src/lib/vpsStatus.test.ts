import { describe, expect, it } from 'vitest';
import { getVpsHostStatus } from './vpsStatus';

describe('getVpsHostStatus', () => {
  it('marks a new dedicated server as pending', () => {
    expect(getVpsHostStatus({ status: 'pending' })).toBe('pending');
  });

  it('marks registered dedicated server hosts as active regardless of agent or mc state', () => {
    for (const status of ['offline', 'online', 'starting', 'stopping', 'deploying', 'custom']) {
      expect(getVpsHostStatus({ status })).toBe('active');
    }
  });

  it('surfaces host-level errors', () => {
    expect(getVpsHostStatus({ status: 'error' })).toBe('error');
  });
});
