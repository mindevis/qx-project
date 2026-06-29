import { describe, expect, it } from 'vitest';
import { isModOnServer, modSupportsServerSync, modSyncSide } from './modSync';

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
