import { useEffect, useState } from 'react';
import {
  Link,
  Navigate,
  NavLink,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams,
} from 'react-router-dom';
import { Spin, Tag, Typography } from 'antd';
import {
  AppstoreOutlined,
  ArrowLeftOutlined,
  DatabaseOutlined,
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
  const base = `/launcher/instances/${instanceId}/resources`;
  const catalogActive = location.pathname.includes('/catalog');

  return (
    <nav className="launcher-resources-tabs" aria-label={t('launcherInstanceResources.tabsAria')}>
      <NavLink
        to={base}
        end
        className={() =>
          `launcher-resources-tab${!catalogActive ? ' launcher-resources-tab--active' : ''}`
        }
      >
        <UnorderedListOutlined aria-hidden />
        {t('launcherInstanceResources.tabInstalled')}
      </NavLink>
      <NavLink
        to={`${base}/catalog`}
        className={() =>
          `launcher-resources-tab${catalogActive ? ' launcher-resources-tab--active' : ''}`
        }
      >
        <AppstoreOutlined aria-hidden />
        {t('launcherInstanceResources.tabCatalog')}
      </NavLink>
    </nav>
  );
}

export function LauncherInstanceResourcesPage() {
  const { t } = useI18n();
  const message = useMessage();
  const navigate = useNavigate();
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

  return (
    <div className="launcher-resources-page">
      <section className="launcher-resources-hero">
        <div className="launcher-resources-hero-ambient" aria-hidden />
        <div className="launcher-resources-hero-inner">
          <Link to="/launcher" className="launcher-resources-back">
            <ArrowLeftOutlined /> {t('launcherInstanceResources.backToLauncher')}
          </Link>
          <div className="launcher-resources-hero-head">
            <div className="launcher-resources-hero-main">
              <span className="launcher-resources-badge">
                <DatabaseOutlined aria-hidden />
                {t('launcherInstanceResources.badge')}
              </span>
              <Title level={1} className="launcher-resources-title">
                {instance.name}
              </Title>
              <Paragraph type="secondary" className="launcher-resources-subtitle">
                {t('launcherInstanceResources.subtitle', {
                  mc: instance.mc_version,
                  loader: loaderName,
                })}
              </Paragraph>
              <div className="launcher-resources-tags">
                <span className="launcher-tag launcher-tag--version">
                  Minecraft {instance.mc_version}
                </span>
                <Tag>{loaderName}</Tag>
                {instance.loader_version ? <Tag>{instance.loader_version}</Tag> : null}
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="launcher-resources-body">
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
