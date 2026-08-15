import { describe, expect, it } from 'vitest';
import type { LauncherInstance } from '@/api/client';
import {
  findCompatibleInstance,
  launcherSpecForMonitoringServer,
  offlineUsernameFromIdentity,
} from './monitoringConnect';

const instance = (partial: Partial<LauncherInstance>): LauncherInstance => ({
  id: 'inst-1',
  name: 'Forge Client',
  mc_version: '1.21.1',
  loader: 'forge',
  loader_version: '47.0.0',
  created_at: 'now',
  updated_at: 'now',
  ...partial,
});

describe('monitoringConnect', () => {
  it('maps modded servers to the same launcher loader', () => {
    expect(
      launcherSpecForMonitoringServer({
        name: 'Neo Survival',
        server_type: 'neoforge',
        mc_version: '1.21.1',
        loader_version: '21.1.234',
      }),
    ).toEqual({
      name: 'Neo Survival',
      mc_version: '1.21.1',
      loader: 'neoforge',
      loader_version: '21.1.234',
    });
  });

  it('maps plugin servers to a vanilla client', () => {
    expect(
      launcherSpecForMonitoringServer({
        name: 'Paper Town',
        server_type: 'paper',
        mc_version: '1.21',
        loader_version: '456',
      }),
    ).toEqual({
      name: 'Paper Town',
      mc_version: '1.21',
      loader: 'vanilla',
    });
  });

  it('rejects modded servers without a loader version', () => {
    expect(
      launcherSpecForMonitoringServer({
        name: 'Broken Forge',
        server_type: 'forge',
        mc_version: '1.21',
      }),
    ).toBeNull();
  });

  it('finds a compatible instance by mc, loader and loader version', () => {
    const items = [
      instance({ id: 'other', mc_version: '1.20.1' }),
      instance({ id: 'match', loader_version: '47.0.0' }),
    ];
    expect(
      findCompatibleInstance(items, {
        name: 'Forge',
        mc_version: '1.21.1',
        loader: 'forge',
        loader_version: '47.0.0',
      })?.id,
    ).toBe('match');
  });

  it('builds an offline username from email when profile name is missing', () => {
    expect(offlineUsernameFromIdentity({ id: '1', email: 'steve@test.com', tier: 'free', created_at: 'now' })).toBe(
      'steve',
    );
    expect(offlineUsernameFromIdentity(null)).toBe('Player');
  });
});
