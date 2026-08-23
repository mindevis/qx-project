import { describe, expect, it } from 'vitest';
import {
  catalogLoaderForType,
  launcherCatalogTabs,
  launcherSupportsDatapacks,
  launcherSupportsModsCatalog,
  launcherSupportsResourcesPage,
} from './launcherInstanceCapabilities';

describe('launcherInstanceCapabilities', () => {
  it('enables mods catalog only for modded loaders', () => {
    expect(launcherSupportsModsCatalog('forge')).toBe(true);
    expect(launcherSupportsModsCatalog('vanilla')).toBe(false);
  });

  it('enables datapacks for all loaders', () => {
    expect(launcherSupportsDatapacks('vanilla')).toBe(true);
    expect(launcherSupportsDatapacks('forge')).toBe(true);
  });

  it('allows resources page for vanilla (datapacks) and modded instances', () => {
    expect(launcherSupportsResourcesPage('vanilla')).toBe(true);
    expect(launcherSupportsResourcesPage('fabric')).toBe(true);
  });

  it('builds catalog tabs per loader', () => {
    expect(launcherCatalogTabs('vanilla')).toEqual(['datapack']);
    expect(launcherCatalogTabs('forge')).toEqual(['mod', 'resourcepack', 'shader', 'datapack']);
  });

  it('uses datapack loader for datapack versions and omits it for packs/shaders', () => {
    expect(catalogLoaderForType('forge', 'datapack')).toBe('datapack');
    expect(catalogLoaderForType('forge', 'resourcepack')).toBeUndefined();
    expect(catalogLoaderForType('forge', 'shader')).toBeUndefined();
    expect(catalogLoaderForType('forge', 'mod')).toBe('forge');
  });
});
