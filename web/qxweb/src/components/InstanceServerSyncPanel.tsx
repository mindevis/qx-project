import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { Button, Modal, Select, Spin, Tag } from 'antd';
import { CloudSyncOutlined } from '@ant-design/icons';
import { api, type InstanceResource } from '@/api/client';
import { ModSyncModal, type ModSyncSelection } from '@/components/ModSyncModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import {
  gameServerSyncTargetKey,
  loadGameServerSyncTargets,
  type GameServerSyncTarget,
} from '@/lib/gameServerSyncTargets';
import {
  instanceResourceSupportsServerSync,
  isInstanceResourceOnServer,
} from '@/lib/modSync';
import { modalMotionProps } from '@/lib/modal';
import { restartVpsGameServer } from '@/lib/vpsGameServers';
import './InstanceServerSyncPanel.css';

type InstanceServerSyncContextValue = {
  canSync: boolean;
  loadingTargets: boolean;
  syncing: boolean;
  targets: GameServerSyncTarget[];
  selectedKey?: string;
  setSelectedKey: (key: string) => void;
  selectedTarget?: GameServerSyncTarget;
  syncableItems: InstanceResource[];
  pendingItems: InstanceResource[];
  syncedCount: number;
  isOnServer: (item: InstanceResource) => boolean;
  syncAll: () => Promise<void>;
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
  const [loadingTargets, setLoadingTargets] = useState(false);
  const [targets, setTargets] = useState<GameServerSyncTarget[]>([]);
  const [selectedKey, setSelectedKey] = useState<string>();
  const [syncing, setSyncing] = useState(false);
  const [singleSyncOpen, setSingleSyncOpen] = useState(false);
  const [singleSelection, setSingleSelection] = useState<ModSyncSelection | null>(null);

  const loadTargets = useCallback(async () => {
    if (!canSync) {
      setTargets([]);
      setSelectedKey(undefined);
      return;
    }
    setLoadingTargets(true);
    try {
      const loaded = await loadGameServerSyncTargets(instance.loader, instance.mc_version);
      setTargets(loaded);
      setSelectedKey((prev) => {
        if (prev && loaded.some((item) => gameServerSyncTargetKey(item) === prev)) {
          return prev;
        }
        return loaded[0] ? gameServerSyncTargetKey(loaded[0]) : undefined;
      });
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('qxmods.sync.loadFailed'));
      setTargets([]);
      setSelectedKey(undefined);
    } finally {
      setLoadingTargets(false);
    }
  }, [canSync, instance.loader, instance.mc_version, message, t]);

  useEffect(() => {
    void loadTargets();
  }, [loadTargets]);

  const selectedTarget = useMemo(
    () => targets.find((item) => gameServerSyncTargetKey(item) === selectedKey),
    [selectedKey, targets],
  );

  const syncableItems = useMemo(
    () => items.filter(instanceResourceSupportsServerSync),
    [items],
  );

  const serverMods = selectedTarget?.serverMods ?? [];

  const isOnServer = useCallback(
    (item: InstanceResource) => isInstanceResourceOnServer(serverMods, item),
    [serverMods],
  );

  const pendingItems = useMemo(
    () => syncableItems.filter((item) => !isOnServer(item)),
    [isOnServer, syncableItems],
  );

  const syncedCount = syncableItems.length - pendingItems.length;

  const refreshServerMods = useCallback(
    async (target: GameServerSyncTarget) => {
      const modsRes = await api.listVpsGameServerMods(target.vpsId, target.gameServer.id);
      const nextMods = modsRes.items ?? [];
      setTargets((prev) =>
        prev.map((item) =>
          gameServerSyncTargetKey(item) === gameServerSyncTargetKey(target)
            ? { ...item, serverMods: nextMods }
            : item,
        ),
      );
    },
    [],
  );

  const promptServerRestart = (target: GameServerSyncTarget) => {
    Modal.confirm({
      ...modalMotionProps,
      title: t('qxmods.sync.restartTitle'),
      content: t('qxmods.sync.restartBulkPrompt', { name: target.gameServer.name }),
      okText: t('qxmods.sync.restartConfirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await restartVpsGameServer(target.vpsId, target.gameServer.id);
          message.success(t('servers.gameServerRestartStarted'));
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('common.error'));
          throw e;
        }
      },
    });
  };

  const syncResources = useCallback(
    async (resources: InstanceResource[]) => {
      if (!selectedTarget || resources.length === 0) return;
      const target = selectedTarget;
      setSyncing(true);
      let queued = 0;
      let failed = 0;

      try {
        for (const resource of resources) {
          if (!resource.project_id || !resource.version_id) continue;
          try {
            const version = await api.getModVersion(
              resource.source,
              resource.project_id,
              resource.version_id,
              { loader: instance.loader, mc_version: instance.mc_version },
            );
            const file = version.files[0];
            if (!file?.url) {
              failed += 1;
              continue;
            }
            const res = await api.syncModToGameServer(target.vpsId, target.gameServer.id, {
              source: resource.source,
              project_id: resource.project_id,
              version_id: resource.version_id,
              filename: file.filename,
              download_url: file.url,
              project_name: resource.project_name,
              version_number: resource.version_number,
            });
            if (res.status !== 'already_installed') {
              queued += 1;
            }
          } catch {
            failed += 1;
          }
        }

        try {
          await refreshServerMods(target);
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('qxmods.sync.statusLoadFailed'));
        }

        if (queued > 0) {
          message.success(t('qxmods.sync.bulkQueued', { count: queued }));
          promptServerRestart(target);
        } else if (failed > 0) {
          message.error(t('qxmods.sync.bulkFailed'));
        } else {
          message.info(t('qxmods.sync.allOnServer'));
        }
      } finally {
        setSyncing(false);
      }
    },
    [instance.loader, instance.mc_version, message, refreshServerMods, selectedTarget, t],
  );

  const syncAll = useCallback(async () => {
    await syncResources(pendingItems);
  }, [pendingItems, syncResources]);

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
      loadingTargets,
      syncing,
      targets,
      selectedKey,
      setSelectedKey,
      selectedTarget,
      syncableItems,
      pendingItems,
      syncedCount,
      isOnServer,
      syncAll,
      openSingleSync,
    }),
    [
      canSync,
      isOnServer,
      loadingTargets,
      openSingleSync,
      pendingItems,
      selectedKey,
      selectedTarget,
      syncAll,
      syncedCount,
      syncableItems,
      syncing,
      targets,
    ],
  );

  return (
    <InstanceServerSyncContext.Provider value={value}>
      {children}
      <ModSyncModal
        open={singleSyncOpen}
        selection={singleSelection}
        instanceLoader={instance.loader}
        onClose={() => {
          setSingleSyncOpen(false);
          setSingleSelection(null);
          if (selectedTarget) {
            void refreshServerMods(selectedTarget).catch(() => undefined);
          }
        }}
      />
    </InstanceServerSyncContext.Provider>
  );
}

export function InstanceServerSyncToolbar() {
  const { t } = useI18n();
  const { openAuthModal } = useAuthModal();
  const {
    canSync,
    loadingTargets,
    syncing,
    targets,
    selectedKey,
    setSelectedKey,
    selectedTarget,
    pendingItems,
    syncableItems,
    syncAll,
  } = useInstanceServerSync();

  if (!canSync) {
    return (
      <Button icon={<CloudSyncOutlined />} onClick={() => openAuthModal('login')}>
        {t('qxmods.sync.action')}
      </Button>
    );
  }

  if (loadingTargets) {
    return <Spin size="small" />;
  }

  if (targets.length === 0) {
    return (
      <Tag className="instance-server-sync-toolbar-tag">{t('qxmods.sync.noServers')}</Tag>
    );
  }

  return (
    <>
      <Select
        showSearch
        className="instance-server-sync-select instance-server-sync-select--toolbar"
        placeholder={t('qxmods.sync.serverPlaceholder')}
        value={selectedKey}
        disabled={syncing}
        optionFilterProp="label"
        options={targets.map((target) => ({
          value: gameServerSyncTargetKey(target),
          label: `${target.gameServer.name} (${target.vpsName})`,
        }))}
        onChange={(value) => setSelectedKey(value)}
      />
      <Button
        type="primary"
        icon={<CloudSyncOutlined />}
        loading={syncing}
        disabled={!selectedTarget || pendingItems.length === 0 || syncableItems.length === 0}
        onClick={() => void syncAll()}
      >
        {t('qxmods.sync.action')}
      </Button>
    </>
  );
}

export function InstanceServerSyncStatus() {
  const { t } = useI18n();
  const {
    canSync,
    loadingTargets,
    targets,
    selectedTarget,
    syncableItems,
    pendingItems,
    syncedCount,
  } = useInstanceServerSync();

  if (!canSync || loadingTargets) {
    return null;
  }

  const statusTag =
    syncableItems.length === 0 ? (
      <Tag>{t('qxmods.sync.noSyncableMods')}</Tag>
    ) : targets.length === 0 ? null : !selectedTarget ? (
      <Tag>{t('qxmods.sync.pickServer')}</Tag>
    ) : pendingItems.length === 0 ? (
      <Tag color="success">{t('qxmods.sync.allOnServer')}</Tag>
    ) : (
      <Tag color="warning">
        {t('qxmods.sync.pendingCount', {
          pending: pendingItems.length,
          total: syncableItems.length,
        })}
      </Tag>
    );

  if (!statusTag && !(selectedTarget && syncableItems.length > 0)) {
    return null;
  }

  return (
    <div className="instance-server-sync-status" aria-live="polite">
      {statusTag}
      {selectedTarget && syncableItems.length > 0 ? (
        <span className="instance-server-sync-summary">
          {t('qxmods.sync.summary', {
            synced: syncedCount,
            total: syncableItems.length,
            server: selectedTarget.gameServer.name,
          })}
        </span>
      ) : null}
    </div>
  );
}

export function InstanceResourceSyncButton({ item }: { item: InstanceResource }) {
  const { t } = useI18n();
  const { openAuthModal } = useAuthModal();
  const { canSync, isOnServer, openSingleSync } = useInstanceServerSync();

  if (!instanceResourceSupportsServerSync(item)) {
    return null;
  }

  const onServer = isOnServer(item);

  return (
    <Button
      type="text"
      size="small"
      className="launcher-resource-card-sync"
      icon={<CloudSyncOutlined />}
      disabled={onServer}
      aria-label={onServer ? t('qxmods.sync.alreadyOnServer') : t('qxmods.sync.withServer')}
      title={onServer ? t('qxmods.sync.alreadyOnServer') : t('qxmods.sync.withServer')}
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

/** @deprecated Use InstanceServerSyncProvider + toolbar components instead. */
export function InstanceServerSyncPanel({ items }: { items: InstanceResource[] }) {
  const { t } = useI18n();
  return (
    <InstanceServerSyncProvider items={items}>
      <div className="instance-server-sync-panel" aria-label={t('qxmods.sync.panelAria')}>
        <InstanceServerSyncStatus />
        <div className="instance-server-sync-actions">
          <InstanceServerSyncToolbar />
        </div>
      </div>
    </InstanceServerSyncProvider>
  );
}
