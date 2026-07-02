import { describe, expect, it } from 'vitest';
import {
  instanceResourceVersionKey,
  isInstanceResourceOnServer,
  isModOnServer,
  modSupportsServerSync,
  modSyncSide,
} from './modSync';

describe('modSyncSide', () => {
  it('detects server-side mods', () => {
    expect(modSyncSide({ client_side: 'unsupported', server_side: 'required' })).toBe('server');
    expect(modSupportsServerSync({ client_side: 'unsupported', server_side: 'required' })).toBe(true);
  });

  it('detects client-only mods', () => {
    expect(modSyncSide({ client_side: 'required', server_side: 'unsupported' })).toBe('client');
    expect(modSupportsServerSync({ client_side: 'required', server_side: 'unsupported' })).toBe(false);
  });
});

describe('isModOnServer', () => {
  it('matches filename on server', () => {
    const onServer = isModOnServer(
      [{ name: 'sodium-0.5.0.jar', path: 'mods/sodium-0.5.0.jar', dir: false }],
      { files: [{ filename: 'sodium-0.5.0.jar', url: 'https://example/mod.jar' }] },
    );
    expect(onServer).toBe(true);
  });

  it('is case-insensitive', () => {
    const onServer = isModOnServer(
      [{ name: 'Mod.JAR', path: 'mods/Mod.JAR', dir: false }],
      { files: [{ filename: 'mod.jar', url: 'https://example/mod.jar' }] },
    );
    expect(onServer).toBe(true);
  });
});

describe('instanceResourceVersionKey', () => {
  it('builds a stable cache key', () => {
    expect(
      instanceResourceVersionKey({
        source: 'curseforge',
        project_id: 'journeymap',
        version_id: 'ver-1',
      }),
    ).toBe('curseforge:journeymap:ver-1');
  });
});

describe('isInstanceResourceOnServer', () => {
  const serverFiles = [{ name: 'actual-mod-file.jar', path: 'mods/actual-mod-file.jar', dir: false }];

  it('matches instance filename by default', () => {
    expect(
      isInstanceResourceOnServer(serverFiles, { filename: 'actual-mod-file.jar' }),
    ).toBe(true);
  });

  it('matches any cached version filename instead of instance filename', () => {
    expect(
      isInstanceResourceOnServer(
        serverFiles,
        { filename: 'launcher-local-name.jar' },
        ['actual-mod-file.jar'],
      ),
    ).toBe(true);
    expect(
      isInstanceResourceOnServer(
        serverFiles,
        { filename: 'launcher-local-name.jar' },
        ['other-file.jar'],
      ),
    ).toBe(false);
  });
});
