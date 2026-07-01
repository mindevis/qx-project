import { useEffect, useState } from 'react';
import { Link, Navigate, Outlet, Route, Routes, useNavigate, useParams } from 'react-router-dom';
import { Spin, Tag, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { api, type LauncherInstance } from '@/api/client';
import { InstanceModsProvider } from '@/components/InstanceModsContext';
import { InstanceInstalledResources } from '@/components/InstanceInstalledResources';
import { ModsCatalogPanel } from '@/components/ModsCatalogPanel';
import { ModDetailPanel } from '@/components/ModDetailPanel';
import { useAuth } from '@/auth/AuthContext';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { isModdedLauncherLoader } from '@/lib/isModdedLoader';
import { isLauncherLoader, type LauncherLoader } from '@/lib/launcherLoaders';
import { logger } from '@/lib/logger';
import './LauncherPage.css';

const { Title, Paragraph } = Typography;

function InstanceModsShell({ instance }: { instance: LauncherInstance }) {
  const { isAuthenticated } = useAuth();
  return (
    <InstanceModsProvider instance={instance} canSync={isAuthenticated}>
      <Outlet />
    </InstanceModsProvider>
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
        if (!isModdedLauncherLoader(found.loader)) {
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
      <div className="launcher-page launcher-page--qxmods">
        <div className="launcher-panel-loading">
          <Spin size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="launcher-page launcher-page--instance-detail launcher-page--qxmods">
      <section className="launcher-section launcher-section--hero launcher-section--qxmods-compact">
        <div className="launcher-hero-inner launcher-hero-inner--qxmods">
          <div className="launcher-hero-content">
            <Link to="/launcher" className="launcher-instance-detail-back">
              <ArrowLeftOutlined /> {t('launcherInstanceResources.backToLauncher')}
            </Link>
            <span className="launcher-section-eyebrow">{t('launcherInstanceResources.badge')}</span>
            <Title level={1} className="launcher-title">
              {instance.name}
            </Title>
            <div className="launcher-instance-tags launcher-instance-tags--detail">
              <span className="launcher-tag launcher-tag--version">
                Minecraft {instance.mc_version}
              </span>
              <Tag>
                {isLauncherLoader(instance.loader)
                  ? loaderLabel(instance.loader)
                  : instance.loader}
              </Tag>
              {instance.loader_version ? <Tag>{instance.loader_version}</Tag> : null}
            </div>
            <Paragraph type="secondary" className="launcher-intro">
              {t('qxmods.promoBody')}
            </Paragraph>
          </div>
        </div>
      </section>

      <section className="launcher-section launcher-section--instance-resources">
        <div className="launcher-panel launcher-panel--resources launcher-panel--qxmods-full">
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
