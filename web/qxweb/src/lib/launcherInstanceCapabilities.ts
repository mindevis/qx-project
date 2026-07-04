import type { ModProjectType } from '@/api/client';
import { isModdedLauncherLoader } from '@/lib/isModdedLoader';

export function launcherSupportsModsCatalog(loader: string): boolean {
  return isModdedLauncherLoader(loader);
}

export function launcherSupportsDatapacks(_loader: string): boolean {
  return true;
}

export function launcherSupportsResourcesPage(loader: string): boolean {
  return launcherSupportsModsCatalog(loader) || launcherSupportsDatapacks(loader);
}

export function launcherCatalogTabs(loader: string): ModProjectType[] {
  const tabs: ModProjectType[] = [];
  if (launcherSupportsModsCatalog(loader)) {
    tabs.push('mod', 'resourcepack', 'shader');
  }
  if (launcherSupportsDatapacks(loader)) {
    tabs.push('datapack');
  }
  return tabs;
}

export function catalogLoaderForType(
  loader: string,
  projectType: ModProjectType,
): string | undefined {
  switch (projectType) {
    case 'datapack':
    case 'resourcepack':
    case 'shader':
      return undefined;
    default:
      return loader;
  }
}
