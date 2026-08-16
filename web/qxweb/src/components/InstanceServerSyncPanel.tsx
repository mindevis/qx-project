import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { Button, Tooltip } from 'antd';
import { CheckCircleOutlined, CloudSyncOutlined } from '@ant-design/icons';
import { api, type InstanceResource, type ModVersion } from '@/api/client';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { useModal } from '@/hooks/useModal';
import {
  loadGameServerSyncTargets,
  type GameServerSyncTarget,
} from '@/lib/gameServerSyncTargets';
import {
  instanceResourceSupportsServerSync,
  instanceResourceVersionKey,
  instanceResourceContentTarget,
  isInstanceResourceOnServer,
  isUploadedInstanceResource,
  mergeServerModLists,
} from '@/lib/modSync';
import './InstanceServerSyncPanel.css';

type InstanceServerSyncContextValue = {
  canSync: boolean;
  targets: GameServerSyncTarget[];
  getSyncedTargets: (item: InstanceResource) => GameServerSyncTarget[];
  openSingleSync: (item: InstanceResource) => void;
  offerRemoveFromServerMods: (item: InstanceResource) => void;
};

const InstanceServerSyncContext = createContext<InstanceServerSyncContextValue | null>(null);

async function resolveModVersionForSync(
  item: InstanceResource,
  loader: string,
  mcVersion?: string,
): Promise<ModVersion | null> {
  if (!item.project_id || !item.version_id) return null;

  const params = { loader, mc_version: mcVersion };
  try {
    return await api.getModVersion(item.source, item.project_id, item.version_id, params);
  } catch {
    try {
      const listed = await api.listModVersions(item.source, item.project_id, params);
      const match = listed.items.find((entry) => entry.id === item.version_id);
      if (match?.files.some((file) => file.url)) {
        return match;
      }
    } catch {
      // Fall back to instance metadata below.
    }
  }

  if (!item.filename) return null;
  return {
    id: item.version_id,
    version_number: item.version_number ?? item.version_id,
    files: [{ filename: item.filename, url: '' }],
  };
}

export function useInstanceServerSync() {
  const ctx = useContext(InstanceServerSyncContext);
  if (!ctx) {
    throw new Error('useInstanceServerSync must be used within InstanceServerSyncProvider');
  }
  return ctx;
}

export function InstanceServerSyncProvider({
  items,
  children,
}: {
  items: InstanceResource[];
  children: ReactNode;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const modal = useModal();
  const { instance, canSync } = useInstanceMods();
  const [targets, setTargets] = useState<GameServerSyncTarget[]>([]);
  const [versionFilenames, setVersionFilenames] = useState<Record<string, string[]>>({});
  const [singleSyncOpen, setSingleSyncOpen] = useState(false);
  const [singleSelection, setSingleSelection] = useState<ModSyncSelection | null>(null);
  const [uploadedSyncResource, setUploadedSyncResource] = useState<InstanceResource | null>(null);
  const [syncContentTarget, setSyncContentTarget] = useState<import('@/lib/modSync').ContentTarget>('mods');
  const [syncResourceType, setSyncResourceType] = useState<'mod' | 'resourcepack' | 'shader'>('mod');

  const syncableItems = useMemo(
    () => items.filter(instanceResourceSupportsServerSync),
    [items],
  );

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      if (!canSync) {
        if (!cancelled) {
          setTargets([]);
        }
        return;
      }
      try {
        const loaded = await loadGameServerSyncTargets(instance.loader, instance.mc_version);
        if (!cancelled) {
          setTargets(loaded);
        }
      } catch (e) {
        if (!cancelled) {
          message.error(e instanceof Error ? e.message : t('qxmods.sync.loadFailed'));
          setTargets([]);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canSync, instance.loader, instance.mc_version, message, t]);

  useEffect(() => {
    if (!canSync || syncableItems.length === 0) {
      setVersionFilenames({});
      return;
    }

    let cancelled = false;

    void (async () => {
      const next: Record<string, string[]> = {};
      await Promise.all(
        syncableItems.map(async (resource) => {
          const key = instanceResourceVersionKey(resource);
          if (!key) return;
          if (isUploadedInstanceResource(resource)) {
            next[key] = [resource.filename];
            return;
          }
          if (!resource.project_id || !resource.version_id) return;
          try {
            const version = await resolveModVersionForSync(resource, instance.loader, instance.mc_version);
            if (version?.files.length) {
              next[key] = version.files.map((file) => file.filename);
            } else if (resource.filename) {
              next[key] = [resource.filename];
            }
          } catch {
            if (resource.filename) {
              next[key] = [resource.filename];
            }
          }
        }),
      );
      if (!cancelled) {
        setVersionFilenames(next);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canSync, instance.loader, instance.mc_version, syncableItems]);

  const isResourceOnTarget = useCallback(
    (item: InstanceResource, target: GameServerSyncTarget) => {
      const key = instanceResourceVersionKey(item);
      const filenames = key ? versionFilenames[key] : undefined;
      return isInstanceResourceOnServer(
        mergeServerModLists(target.serverMods, target.clientMods ?? []),
        item,
        filenames,
      );
    },
    [versionFilenames],
  );

  const getSyncedTargets = useCallback(
    (item: InstanceResource) => targets.filter((target) => isResourceOnTarget(item, target)),
    [isResourceOnTarget, targets],
  );

  const refreshAllServerMods = useCallback(async () => {
    if (targets.length === 0) return;
    const refreshed = await Promise.all(
      targets.map(async (target) => {
        try {
          const modsRes = await api.listVpsGameServerMods(target.vpsId, target.gameServer.id);
          let clientMods: Awaited<ReturnType<typeof api.listVpsGameServerClientMods>>['items'] = [];
          try {
            const clientRes = await api.listVpsGameServerClientMods(target.vpsId, target.gameServer.id);
            clientMods = clientRes.items ?? [];
          } catch {
            clientMods = [];
          }
          return { ...target, serverMods: modsRes.items ?? [], clientMods };
        } catch {
          return target;
        }
      }),
    );
    setTargets(refreshed);
  }, [targets]);

  const openSingleSync = useCallback(
    async (item: InstanceResource) => {
      if (isUploadedInstanceResource(item)) {
        setUploadedSyncResource(item);
        setSingleSelection(null);
        setSyncContentTarget(instanceResourceContentTarget(item));
        setSyncResourceType(
          item.resource_type === 'resourcepack' || item.resource_type === 'shader'
            ? item.resource_type
            : 'mod',
        );
        setSingleSyncOpen(true);
        return;
      }
      if (!item.project_id || !item.version_id) return;
      const version = await resolveModVersionForSync(item, instance.loader, instance.mc_version);
      if (!version?.files[0]?.url) {
        message.error(t('qxmods.sync.failed'));
        return;
      }
      const versionKey = instanceResourceVersionKey(item);
      if (versionKey) {
        setVersionFilenames((prev) => ({
          ...prev,
          [versionKey]: version.files.map((entry) => entry.filename),
        }));
      }
      setSingleSelection({
        source: item.source,
        projectId: item.project_id,
        projectName: item.project_name,
        version,
      });
      setSyncContentTarget(instanceResourceContentTarget(item));
      setSyncResourceType(
        item.resource_type === 'resourcepack' || item.resource_type === 'shader'
          ? item.resource_type
          : 'mod',
      );
      setUploadedSyncResource(null);
      setSingleSyncOpen(true);
    },
    [instance.loader, instance.mc_version, message, t],
  );

  const offerRemoveFromServerMods = useCallback(
    (item: InstanceResource) => {
      if (item.resource_type !== 'mod') return;
      const key = instanceResourceVersionKey(item);
      const filenames = key ? versionFilenames[key] : undefined;
      const affected = targets.filter((target) =>
        isInstanceResourceOnServer(target.serverMods, item, filenames),
      );
      if (affected.length === 0) return;
      modal.confirm({
        title: t('qxmods.side.serverCleanupTitle'),
        content: t('qxmods.side.serverCleanupBody', {
          name: item.project_name,
          servers: affected.map((target) => target.gameServer.name).join(', '),
        }),
        okText: t('qxmods.side.serverCleanupConfirm'),
        cancelText: t('common.cancel'),
        okButtonProps: { danger: true },
        onOk: async () => {
          let failed = false;
          await Promise.all(
            affected.map(async (target) => {
              try {
                await api.deleteVpsGameServerMod(target.vpsId, target.gameServer.id, {
                  filename: item.filename,
                  mod_target: 'mods',
                });
              } catch {
                failed = true;
              }
            }),
          );
          if (failed) {
            message.error(t('qxmods.side.serverCleanupFailed'));
          } else {
            message.success(t('qxmods.side.serverCleanupDone'));
          }
          await refreshAllServerMods();
        },
      });
    },
    [message, modal, refreshAllServerMods, t, targets, versionFilenames],
  );

  const value = useMemo(
    () => ({
      canSync,
      targets,
      getSyncedTargets,
      openSingleSync,
      offerRemoveFromServerMods,
    }),
    [canSync, getSyncedTargets, offerRemoveFromServerMods, openSingleSync, targets],
  );

  return (
    <InstanceServerSyncContext.Provider value={value}>
      {children}
      <ModSyncModal
        open={singleSyncOpen}
        selection={singleSelection}
        uploadedResource={uploadedSyncResource}
        instanceId={instance.id}
        instanceLoader={instance.loader}
        instanceMcVersion={instance.mc_version}
        installedResources={items}
        modTarget={syncContentTarget}
        resourceType={syncResourceType}
        onClose={() => {
          setSingleSyncOpen(false);
          setSingleSelection(null);
          setUploadedSyncResource(null);
          void refreshAllServerMods();
        }}
      />
    </InstanceServerSyncContext.Provider>
  );
}

export function InstanceResourceSyncButton({ item }: { item: InstanceResource }) {
  const { t } = useI18n();
  const { openAuthModal } = useAuthModal();
  const { canSync, getSyncedTargets, openSingleSync } = useInstanceServerSync();

  if (!instanceResourceSupportsServerSync(item)) {
    return null;
  }

  const syncedTargets = getSyncedTargets(item);
  const syncedServerNames = syncedTargets.map((target) => target.gameServer.name);

  if (syncedTargets.length > 0) {
    const tooltip =
      syncedServerNames.length === 1
        ? t('qxmods.sync.syncedWithOne', { server: syncedServerNames[0] })
        : t('qxmods.sync.syncedWith', { servers: syncedServerNames.join(', ') });

    return (
      <Tooltip title={tooltip}>
        <span className="launcher-resource-card-sync launcher-resource-card-sync--synced" aria-label={tooltip}>
          <CheckCircleOutlined />
        </span>
      </Tooltip>
    );
  }

  return (
    <Button
      type="text"
      size="small"
      className="launcher-resource-card-sync"
      icon={<CloudSyncOutlined />}
      aria-label={t('qxmods.sync.withServer')}
      title={t('qxmods.sync.withServer')}
      onClick={() => {
        if (!canSync) {
          openAuthModal('login');
          return;
        }
        void openSingleSync(item);
      }}
    />
  );
}
