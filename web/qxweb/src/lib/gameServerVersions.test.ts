import { describe, expect, it, vi, afterEach } from 'vitest';
import {
  clearGameServerVersionsCache,
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
    clearGameServerVersionsCache();
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
          return Promise.resolve(
            new Response(
              JSON.stringify({
                versions: {
                  '1.21': ['1.21.11-rc1', '1.21.11', '1.21'],
                  '1.20': ['1.20.4'],
                },
              }),
              { status: 200 },
            ),
          );
        }
        if (url.includes('/versions/1.21.11/builds')) {
          return Promise.resolve(
            new Response(
              JSON.stringify([
                { id: 132, channel: 'STABLE' },
                { id: 66, channel: 'BETA' },
                { id: 48, channel: 'ALPHA' },
              ]),
              { status: 200 },
            ),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mcOptions = await listGameServerMcVersions('paper', mcFallback);
    expect(mcOptions.map((item) => item.value)).toEqual(['1.21.11', '1.21', '1.20.4']);

    const loaderOptions = await listGameServerLoaderVersions('paper', '1.21.11');
    expect(loaderOptions).toEqual([
      { value: '132', label: '#132' },
      { value: '66', label: '#66 (beta)' },
      { value: '48', label: '#48 (alpha)' },
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
        let host: string;
        try {
          host = new URL(url, 'https://example.test').hostname;
        } catch {
          host = '';
        }
        if (url.includes('/upstream/neoforge') || host === 'maven.neoforged.net') {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                isSnapshot: false,
                versions: [
                  '21.1.234',
                  '21.1.233',
                  '21.0.167',
                  '20.6.139',
                  '21.2.1-beta',
                  '26.1.2.76',
                  '26.0.0-beta',
                ],
              }),
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

  it('accepts a raw neoforge version array from older mirrors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/upstream/neoforge')) {
          return Promise.resolve(new Response(JSON.stringify(['21.1.10', '21.1.9']), { status: 200 }));
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const loaderOptions = await listGameServerLoaderVersions('neoforge', '1.21.1');
    expect(loaderOptions.map((item) => item.value)).toEqual(['21.1.10', '21.1.9']);
  });

  it('loads purpur mc and build versions', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/projects/purpur') && !url.includes('/versions/')) {
          return Promise.resolve(new Response(JSON.stringify({ versions: ['1.21'] }), { status: 200 }));
        }
        if (url.includes('/versions/1.21/builds')) {
          return Promise.resolve(
            new Response(JSON.stringify({ builds: [{ build: 100 }] }), { status: 200 }),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mcOptions = await listGameServerMcVersions('purpur', mcFallback);
    expect(mcOptions[0]?.value).toBe('1.21');
    const loaderOptions = await listGameServerLoaderVersions('purpur', '1.21');
    expect(loaderOptions[0]?.value).toBe('100');
  });

  it('loads spigot builds via paper api', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/projects/paper/versions/1.21/builds')) {
          return Promise.resolve(
            new Response(JSON.stringify({ builds: [{ build: 12 }] }), { status: 200 }),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );
    const loaderOptions = await listGameServerLoaderVersions('spigot', '1.21');
    expect(loaderOptions[0]?.value).toBe('12');
  });

  it('loads fabric and quilt compatible loader versions', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/v2/versions/game') || url.includes('/v3/versions/game')) {
          return Promise.resolve(
            new Response(JSON.stringify([{ version: '1.21', stable: true }]), { status: 200 }),
          );
        }
        if (url.includes('/v2/versions/loader') || url.includes('/v3/versions/loader')) {
          return Promise.resolve(
            new Response(JSON.stringify([{ version: '0.15.0' }, { version: '0.14.0' }]), {
              status: 200,
            }),
          );
        }
        if (init?.method === 'HEAD') {
          return Promise.resolve(
            new Response(null, { status: url.includes('/0.15.0/') ? 200 : 404 }),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const fabricMc = await listGameServerMcVersions('fabric', mcFallback);
    expect(fabricMc[0]?.value).toBe('1.21');
    const fabricLoader = await listGameServerLoaderVersions('fabric', '1.21');
    expect(fabricLoader.map((item) => item.value)).toContain('0.15.0');

    const quiltLoader = await listGameServerLoaderVersions('quilt', '1.21');
    expect(quiltLoader.map((item) => item.value)).toContain('0.15.0');
  });

  it('loads mohist and magma versions', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/projects/mohist')) {
          if (url.includes('/builds')) {
            return Promise.resolve(
              new Response(JSON.stringify({ builds: [{ number: 3 }, { number: 2 }] }), {
                status: 200,
              }),
            );
          }
          return Promise.resolve(new Response(JSON.stringify({ versions: ['1.20.1'] }), { status: 200 }));
        }
        if (url.includes('/api/versions')) {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                versions: [
                  { version: '1.20.6-abc', minecraftVersion: '1.20.6' },
                  { version: '1.20.x-legacy', minecraftVersion: '1.20.x' },
                ],
              }),
              { status: 200 },
            ),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mohistMc = await listGameServerMcVersions('mohist', mcFallback);
    expect(mohistMc[0]?.value).toBe('1.20.1');
    const mohistLoader = await listGameServerLoaderVersions('mohist', '1.20.1');
    expect(mohistLoader[0]?.value).toBe('3');

    const magmaMc = await listGameServerMcVersions('magma', mcFallback);
    expect(magmaMc.some((item) => item.value === '1.20.6')).toBe(true);
    const magmaLoader = await listGameServerLoaderVersions('magma', '1.20.6');
    expect(magmaLoader.map((item) => item.value)).toContain('1.20.6-abc');
  });

  it('loads arclight mc and loader versions', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith('/minecraft')) {
          return Promise.resolve(
            new Response(JSON.stringify({ files: [{ name: '1.20.1', type: 'dir' }] }), {
              status: 200,
            }),
          );
        }
        if (url.includes('/loaders') && url.endsWith('/loaders')) {
          return Promise.resolve(
            new Response(JSON.stringify({ files: [{ name: 'forge', type: 'dir' }] }), {
              status: 200,
            }),
          );
        }
        if (url.includes('versions-stable')) {
          return Promise.resolve(
            new Response(JSON.stringify({ files: [{ name: '1.0', type: 'file' }] }), {
              status: 200,
            }),
          );
        }
        if (url.includes('versions-snapshot')) {
          return Promise.reject(new Error('snapshot unavailable'));
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const mcOptions = await listGameServerMcVersions('arclight', mcFallback);
    expect(mcOptions[0]?.value).toBe('1.20.1');
    const loaderOptions = await listGameServerLoaderVersions('arclight', '1.20.1');
    expect(loaderOptions[0]?.value).toBe('forge:1.0');
  });

  it('loads velocity versions and builds from papermc', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/projects/velocity') && !url.includes('/versions/')) {
          return Promise.resolve(
            new Response(JSON.stringify({ versions: { '3.4.0-SNAPSHOT': ['3.4.0-SNAPSHOT'] } }), {
              status: 200,
            }),
          );
        }
        if (url.includes('/projects/velocity/versions/3.4.0-SNAPSHOT/builds')) {
          return Promise.resolve(
            new Response(JSON.stringify([{ id: 550, channel: 'default' }]), { status: 200 }),
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      }),
    );

    const versions = await listGameServerMcVersions('velocity', mcFallback);
    expect(versions[0]).toEqual({ value: '3.4.0-SNAPSHOT', label: '3.4.0-SNAPSHOT' });
    const builds = await listGameServerLoaderVersions('velocity', '3.4.0-SNAPSHOT');
    expect(builds[0]?.value).toBe('550');
  });

  it('returns empty loader list for vanilla', async () => {
    expect(await listGameServerLoaderVersions('vanilla', '1.21')).toEqual([]);
    expect(await listGameServerLoaderVersions('forge', '')).toEqual([]);
    const vanillaMc = await listGameServerMcVersions('vanilla', mcFallback);
    expect(vanillaMc[0]?.value).toBe('1.21');
  });

  it('falls back for unknown server types', async () => {
    const unknownType = 'custom' as Parameters<typeof listGameServerMcVersions>[0];
    const mc = await listGameServerMcVersions(unknownType, mcFallback);
    expect(mc[0]?.value).toBe('1.21');
    const loaders = await listGameServerLoaderVersions(unknownType, '1.21');
    expect(loaders).toEqual([]);
  });

  it('returns empty loader list when upstream throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));
    const loaders = await listGameServerLoaderVersions('paper', '1.21');
    expect(loaders).toEqual([]);
  });
});
