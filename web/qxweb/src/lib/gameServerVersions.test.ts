import { describe, expect, it, vi, afterEach } from 'vitest';
import {
  formatGameServerLoaderVersionLabel,
  formatGameServerMcVersionLabel,
  gameServerTypeNeedsLoader,
  listGameServerLoaderVersions,
  listGameServerMcVersions,
  mcVersionOptionsFromItems,
} from './gameServerVersions';

const mcFallback = [
  { id: '1.21', type: 'release' },
  { id: '1.20.4', type: 'release' },
  { id: '24w14a', type: 'snapshot' },
];

const forgeMavenXml = `<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <versioning>
    <versions>
      <version>1.20.1-47.2.0</version>
      <version>1.20.1-47.3.0</version>
      <version>1.20.1-47.1.0</version>
    </versions>
  </versioning>
</metadata>`;

describe('gameServerVersions', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('detects when loader version is required', () => {
    expect(gameServerTypeNeedsLoader('vanilla')).toBe(false);
    expect(gameServerTypeNeedsLoader('forge')).toBe(true);
  });

  it('builds vanilla mc options from items', () => {
    expect(mcVersionOptionsFromItems(mcFallback).map((item) => item.value)).toEqual(['1.21', '1.20.4']);
  });

  it('formats separate mc and core labels for table', () => {
    expect(formatGameServerMcVersionLabel('1.21')).toBe('1.21');
    expect(formatGameServerLoaderVersionLabel('47.2.0', 'forge')).toBe('47.2.0');
    expect(formatGameServerLoaderVersionLabel(undefined, 'vanilla')).toBe('—');
  });

  it('loads all forge loader versions for selected mc version', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('promotions_slim.json')) {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                promos: {
                  '1.20.1-recommended': '47.2.0',
                  '1.20.1-latest': '47.3.0',
                },
              }),
              { status: 200 },
            ),
          );
        }
        if (url.includes('maven-metadata.xml')) {
          return Promise.resolve(new Response(forgeMavenXml, { status: 200 }));
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mcOptions = await listGameServerMcVersions('forge', mcFallback);
    expect(mcOptions[0]).toEqual({ value: '1.20.1', label: '1.20.1' });

    const loaderOptions = await listGameServerLoaderVersions('forge', '1.20.1');
    expect(loaderOptions.map((item) => item.value)).toEqual(['47.3.0', '47.2.0', '47.1.0']);
    expect(loaderOptions.find((item) => item.value === '47.2.0')?.label).toBe('47.2.0 (recommended)');
    expect(loaderOptions.find((item) => item.value === '47.3.0')?.label).toBe('47.3.0 (latest)');
  });

  it('loads paper builds for selected mc version', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/projects/paper') && !url.includes('/versions/')) {
          return Promise.resolve(new Response(JSON.stringify({ versions: ['1.21'] }), { status: 200 }));
        }
        if (url.includes('/versions/1.21/builds')) {
          return Promise.resolve(
            new Response(JSON.stringify({ builds: [{ build: 456 }, { build: 455 }] }), { status: 200 }),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mcOptions = await listGameServerMcVersions('paper', mcFallback);
    expect(mcOptions[0]?.value).toBe('1.21');

    const loaderOptions = await listGameServerLoaderVersions('paper', '1.21');
    expect(loaderOptions).toEqual([
      { value: '456', label: '#456' },
      { value: '455', label: '#455' },
    ]);
  });

  it('falls back to mc releases when upstream fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));

    const options = await listGameServerMcVersions('paper', mcFallback);
    expect(options[0]?.value).toBe('1.21');
  });

  it('maps neoforge versions to minecraft versions and filters loader builds', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/upstream/neoforge') || url.includes('maven.neoforged.net')) {
          return Promise.resolve(
            new Response(
              JSON.stringify([
                '21.1.234',
                '21.1.233',
                '21.0.167',
                '20.6.139',
                '26.0.0-beta',
              ]),
              { status: 200 },
            ),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mcOptions = await listGameServerMcVersions('neoforge', mcFallback);
    expect(mcOptions.map((item) => item.value)).toEqual(['1.21.1', '1.21', '1.20.6']);

    const loaderOptions = await listGameServerLoaderVersions('neoforge', '1.21.1');
    expect(loaderOptions.map((item) => item.value)).toEqual(['21.1.234', '21.1.233']);
  });
});
