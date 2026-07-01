import { InstanceInstalledResources } from '@/components/InstanceInstalledResources';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import type { LauncherInstance } from '@/api/client';
import { launcherSupportsResourcesPage } from '@/lib/launcherInstanceCapabilities';
import './InstanceResourcesPanel.css';

type InstanceResourcesPanelProps = {
  instance: LauncherInstance;
  canSync: boolean;
  layout?: 'embedded' | 'standalone';
};

/** Embedded resources panel (installed mods + link to catalog). */
export function InstanceResourcesPanel({
  instance,
  canSync,
  layout = 'embedded',
}: InstanceResourcesPanelProps) {
  if (!launcherSupportsResourcesPage(instance.loader)) {
    return null;
  }

  return (
    <InstanceModsProvider instance={instance} canSync={canSync}>
      <InstanceInstalledResources layout={layout} />
    </InstanceModsProvider>
  );
}
