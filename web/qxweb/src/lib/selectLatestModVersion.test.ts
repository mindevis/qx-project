import { describe, expect, it } from 'vitest';
import { selectLatestCompatibleVersion } from './selectLatestModVersion';

const older = {
  id: 'old',
  version_number: '1.0.0',
  game_versions: ['1.21.1'],
  loaders: ['neoforge'],
  files: [],
  published_at: '2025-01-01T00:00:00Z',
};

const newest = {
  id: 'new',
  version_number: '2.0.0',
  game_versions: ['1.21.1'],
  loaders: ['neoforge'],
  files: [],
  published_at: '2026-08-01T00:00:00Z',
};

const fabricOnly = {
  id: 'fabric',
  version_number: '3.0.0',
  game_versions: ['1.21.1'],
  loaders: ['fabric'],
  files: [],
  published_at: '2026-08-10T00:00:00Z',
};

describe('selectLatestCompatibleVersion', () => {
  it('picks the newest matching loader and Minecraft version', () => {
    const latest = selectLatestCompatibleVersion(
      [older, fabricOnly, newest],
      'neoforge',
      '1.21.1',
    );
    expect(latest?.id).toBe('new');
  });

  it('falls back to newest overall when nothing matches filters', () => {
    const latest = selectLatestCompatibleVersion([older, newest], 'quilt', '1.20.1');
    expect(latest?.id).toBe('new');
  });
});
