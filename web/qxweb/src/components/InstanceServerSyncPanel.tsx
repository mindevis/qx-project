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
import { api, type InstanceResource } from '@/api/client';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  loadGameServerSyncTargets,
  type GameServerSyncTarget,
} from '@/lib/gameServerSyncTargets';
import {
  instanceResourceSupportsServerSync,
  instanceResourceVersionKey,
  isInstanceResourceOnServer,
} from '@/lib/modSync';
import './InstanceServerSyncPanel.css';

type InstanceServerSyncContextValue = {
  canSync: boolean;
  targets: GameServerSyncTarget[];
  getSyncedTargets: (item: InstanceResource) => GameServerSyncTarget[];
  openSingleSync: (item: InstanceResource) => void;
};

const InstanceServerSyncContext = createContext<InstanceServerSyncContextValue | null>(null);

function useInstanceServerSync() {
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
  const { instance, canSync } = useInstanceMods();
  const [targets, setTargets] = useState<GameServerSyncTarget[]>([]);
  const [versionFilenames, setVersionFilenames] = useState<Record<string, string[]>>({});
  const [singleSyncOpen, setSingleSyncOpen] = useState(false);
  const [singleSelection, setSingleSelection] = useState<ModSyncSelection | null>(null);

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
          if (!key || !resource.project_id || !resource.version_id) return;
          try {
            const version = await api.getModVersion(
              resource.source,
              resource.project_id,
              resource.version_id,
              { loader: instance.loader, mc_version: instance.mc_version },
            );
            next[key] = version.files.map((file) => file.filename);
          } catch {
            // Fall back to instance filename when version metadata is unavailable.
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
      return isInstanceResourceOnServer(target.serverMods, item, filenames);
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
          return { ...target, serverMods: modsRes.items ?? [] };
        } catch {
          return target;
        }
      }),
    );
    setTargets(refreshed);
  }, [targets]);

  const openSingleSync = useCallback(
    async (item: InstanceResource) => {
      if (!item.project_id || !item.version_id) return;
      try {
        const version = await api.getModVersion(
          item.source,
          item.project_id,
          item.version_id,
          { loader: instance.loader, mc_version: instance.mc_version },
        );
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
        setSingleSyncOpen(true);
      } catch (e) {
        message.error(e instanceof Error ? e.message : t('qxmods.sync.failed'));
      }
    },
    [instance.loader, instance.mc_version, message, t],
  );

  const value = useMemo(
    () => ({
      canSync,
      targets,
      getSyncedTargets,
      openSingleSync,
    }),
    [canSync, getSyncedTargets, openSingleSync, targets],
  );

  return (
    <InstanceServerSyncContext.Provider value={value}>
      {children}
      <ModSyncModal
        open={singleSyncOpen}
        selection={singleSelection}
        instanceLoader={instance.loader}
        instanceMcVersion={instance.mc_version}
        onClose={() => {
          setSingleSyncOpen(false);
          setSingleSelection(null);
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
