import { useEffect, useState } from 'react';
import {
  Link,
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams,
} from 'react-router-dom';
import { Segmented, Spin, Tag, Typography } from 'antd';
import {
  AppstoreOutlined,
  ArrowLeftOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { api, type LauncherInstance } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { InstanceInstalledResources } from '@/components/InstanceInstalledResources';
import { ModsCatalogPanel } from '@/components/ModsCatalogPanel';
import { ModDetailPanel } from '@/components/ModDetailPanel';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { launcherSupportsResourcesPage } from '@/lib/launcherInstanceCapabilities';
import { isLauncherLoader, type LauncherLoader } from '@/lib/launcherLoaders';
import { isServerManagedInstance } from '@/lib/serverManagedInstance';
import { logger } from '@/lib/logger';
import './LauncherInstanceResourcesPage.css';

const { Title, Paragraph } = Typography;

function InstanceModsShell({ instance }: { instance: LauncherInstance }) {
  const { isAuthenticated } = useAuth();
  return (
    <InstanceModsProvider instance={instance} canSync={isAuthenticated}>
      <Outlet />
    </InstanceModsProvider>
  );
}

function ResourcesTabNav({ instanceId }: { instanceId: string }) {
  const { t } = useI18n();
  const location = useLocation();
  const navigate = useNavigate();
  const base = `/launcher/instances/${instanceId}/resources`;
  const catalogActive = location.pathname.includes('/catalog');

  return (
    <nav className="launcher-resources-tabs" aria-label={t('launcherInstanceResources.tabsAria')}>
      <Segmented
        value={catalogActive ? 'catalog' : 'installed'}
        onChange={(value) => {
          navigate(value === 'catalog' ? `${base}/catalog` : base);
        }}
        options={[
          {
            value: 'installed',
            icon: <UnorderedListOutlined aria-hidden />,
            label: t('launcherInstanceResources.tabInstalled'),
          },
          {
            value: 'catalog',
            icon: <AppstoreOutlined aria-hidden />,
            label: t('launcherInstanceResources.tabCatalog'),
          },
        ]}
      />
    </nav>
  );
}

export function LauncherInstanceResourcesPage() {
  const { t } = useI18n();
  const message = useMessage();
  const navigate = useNavigate();
  const location = useLocation();
  const { instanceId } = useParams<{ instanceId: string }>();
  const [instance, setInstance] = useState<LauncherInstance | null>(null);
  const [loading, setLoading] = useState(true);

  const loaderLabel = (loader: LauncherLoader) => {
    const key = `servers.gameServerType.${loader}`;
    const label = t(key);
    return label === key ? loader : label;
  };

  useEffect(() => {
    if (!instanceId) return;

    let cancelled = false;
    setLoading(true);

    void (async () => {
      try {
        const res = await api.listInstances();
        if (cancelled) return;
        const found = res.items?.find((item) => item.id === instanceId) ?? null;
        setInstance(found);
        if (!found) {
          message.error(t('launcherInstanceResources.notFound'));
          navigate('/launcher');
          return;
        }
        if (!launcherSupportsResourcesPage(found.loader)) {
          navigate('/launcher');
          return;
        }
      } catch (e) {
        if (cancelled) return;
        logger.warn('failed to load launcher instance resources', { error: String(e) });
        message.error(t('launcherInstanceResources.loadFailed'));
        navigate('/launcher');
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [instanceId, message, navigate, t]);

  if (loading || !instance) {
    return (
      <div className="launcher-resources-page">
        <div className="launcher-panel-loading" style={{ minHeight: '50vh' }}>
          <Spin size="large" />
        </div>
      </div>
    );
  }

  const loaderName = isLauncherLoader(instance.loader)
    ? loaderLabel(instance.loader)
    : instance.loader;
  const catalogActive = location.pathname.includes('/catalog');

  return (
    <div className={`launcher-resources-page${catalogActive ? ' launcher-resources-page--catalog' : ''}`}>
      <section className="launcher-resources-hero">
        <div className="launcher-resources-hero-inner">
          <Link to="/launcher" className="launcher-resources-back">
            <ArrowLeftOutlined /> {t('launcherInstanceResources.backToLauncher')}
          </Link>
          <div className="launcher-resources-hero-head">
            <div className="launcher-resources-hero-main">
              <Title level={1} className="launcher-resources-title">
                {instance.name}
              </Title>
              <Paragraph type="secondary" className="launcher-resources-subtitle">
                {t('launcherInstanceResources.subtitle', {
                  mc: instance.mc_version,
                  loader: loaderName,
                })}
              </Paragraph>
              {isServerManagedInstance(instance) ? (
                <Paragraph type="warning" className="launcher-resources-subtitle">
                  {t('launcherInstanceResources.managedLocked')}
                </Paragraph>
              ) : null}
              <div className="launcher-resources-tags">
                <span className="launcher-tag launcher-tag--version">
                  Minecraft {instance.mc_version}
                </span>
                <Tag variant="filled">{loaderName}</Tag>
                {instance.loader_version ? <Tag variant="filled">{instance.loader_version}</Tag> : null}
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className={`launcher-resources-body${catalogActive ? ' launcher-resources-body--catalog' : ''}`}>
        <ResourcesTabNav instanceId={instance.id} />
        <div className="launcher-resources-panel">
          <Routes>
            <Route element={<InstanceModsShell instance={instance} />}>
              <Route index element={<InstanceInstalledResources />} />
              <Route path="catalog" element={<ModsCatalogPanel />} />
              <Route path="catalog/:source/:projectId" element={<ModDetailPanel />} />
              <Route path="*" element={<Navigate to="." replace />} />
            </Route>
          </Routes>
        </div>
      </section>
    </div>
  );
}
