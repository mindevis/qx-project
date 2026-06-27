import { useEffect, useState, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Alert, Button, Spin, Typography } from 'antd';
import {
  CheckCircleOutlined,
  CopyOutlined,
  DesktopOutlined,
  ExclamationCircleOutlined,
  LinkOutlined,
  LoginOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  api,
  isBackendUnavailableError,
  saveLinkedDevice,
  type DeviceStatus,
} from '@/api/client';
import { useAuth } from '@/auth/AuthContext';
import { useAuthModal } from '@/auth/AuthModalContext';
import { useMessage } from '@/hooks/useMessage';
import { useI18n } from '@/i18n/I18nContext';
import { logger } from '@/lib/logger';
import './LauncherPage.css';
import './LauncherLinkPage.css';

const { Title, Paragraph, Text } = Typography;

function formatOs(os?: string): string {
  if (!os) return '';
  const labels: Record<string, string> = {
    windows: 'Windows',
    darwin: 'macOS',
    linux: 'Linux',
  };
  return labels[os.toLowerCase()] ?? os;
}

function statusTagClass(status: string): string {
  if (status === 'linked') return 'launcher-link-status-tag--linked';
  if (status === 'pending_link') return 'launcher-link-status-tag--pending';
  return 'launcher-link-status-tag--expired';
}

function DeviceInfoPanel({
  deviceId,
  info,
  loading,
}: {
  deviceId: string;
  info: DeviceStatus | null;
  loading: boolean;
}) {
  const { t, locale } = useI18n();
  const message = useMessage();

  const handleCopyDeviceId = async () => {
    try {
      await navigator.clipboard.writeText(deviceId);
      message.success(t('launcherLink.copied'));
    } catch {
      message.error(t('launcherLink.copyFailed'));
    }
  };

  const formatDateTime = (value?: string) => {
    if (!value) return t('launcherLink.unknown');
    try {
      return new Intl.DateTimeFormat(locale === 'en' ? 'en-US' : 'ru-RU', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(new Date(value));
    } catch {
      return value;
    }
  };

  const statusKey = `launcherLink.deviceStatus.${info?.status ?? 'pending_link'}`;
  const statusLabel = t(statusKey);
  const displayStatus = statusLabel === statusKey ? (info?.status ?? t('launcherLink.unknown')) : statusLabel;

  return (
    <div className="launcher-link-device-band">
      <div className="launcher-link-device-band-head">
        <span className="launcher-link-device-icon">
          <DesktopOutlined />
        </span>
        <div className="launcher-link-device-meta">
          <span className="launcher-link-device-label">{t('launcherLink.deviceLabel')}</span>
          <Text className="launcher-link-device-id" copyable={false}>
            {deviceId}
          </Text>
        </div>
        <Button icon={<CopyOutlined />} onClick={() => void handleCopyDeviceId()}>
          {t('launcherLink.copyId')}
        </Button>
      </div>

      {loading ? (
        <div className="launcher-link-device-loading">
          <Spin tip={t('launcherLink.loadingDevice')} />
        </div>
      ) : (
        <>
          <Text strong style={{ display: 'block', marginBottom: 14 }}>
            {t('launcherLink.deviceInfoTitle')}
          </Text>
          <div className="launcher-link-device-grid">
            <div className="launcher-link-device-row">
              <span className="launcher-link-device-row-label">{t('launcherLink.statusLabel')}</span>
              <span className={`launcher-link-status-tag ${statusTagClass(info?.status ?? '')}`}>
                {displayStatus}
              </span>
            </div>
            <div className="launcher-link-device-row">
              <span className="launcher-link-device-row-label">{t('launcherLink.hostname')}</span>
              <Text className="launcher-link-device-row-value">
                {info?.hostname?.trim() || t('launcherLink.unknown')}
              </Text>
            </div>
            <div className="launcher-link-device-row">
              <span className="launcher-link-device-row-label">{t('launcherLink.os')}</span>
              <Text className="launcher-link-device-row-value">
                {formatOs(info?.os) || t('launcherLink.unknown')}
              </Text>
            </div>
            <div className="launcher-link-device-row">
              <span className="launcher-link-device-row-label">{t('launcherLink.launcherVersion')}</span>
              <Text className="launcher-link-device-row-value">
                {info?.launcher_version?.trim() || t('launcherLink.unknown')}
              </Text>
            </div>
            {info?.link_expires_at && info.status === 'pending_link' && (
              <div className="launcher-link-device-row">
                <span className="launcher-link-device-row-label">{t('launcherLink.linkExpires')}</span>
                <Text className="launcher-link-device-row-value">
                  {formatDateTime(info.link_expires_at)}
                </Text>
              </div>
            )}
            <div className="launcher-link-device-row">
              <span className="launcher-link-device-row-label">{t('launcherLink.lastSeen')}</span>
              <Text className="launcher-link-device-row-value">
                {formatDateTime(info?.last_seen_at)}
              </Text>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function LinkHero({
  variant = 'default',
  icon,
}: {
  variant?: 'default' | 'success' | 'error';
  icon: ReactNode;
}) {
  const { t } = useI18n();
  const heroClass = [
    'launcher-hero',
    variant === 'success' && 'launcher-link-hero--success',
    variant === 'error' && 'launcher-link-hero--error',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <section className={heroClass}>
      <div className="launcher-hero-ambient" aria-hidden>
        <span className="launcher-hero-blob launcher-hero-blob--1" />
        <span className="launcher-hero-blob launcher-hero-blob--2" />
        <span className="launcher-hero-blob launcher-hero-blob--3" />
        <span className="launcher-hero-grid-pattern" />
      </div>

      <div className="launcher-hero-inner">
        <div className="launcher-hero-content">
          <span className="launcher-badge">{t('launcherLink.badge')}</span>
          <Title level={1} className="launcher-title">
            <span className="launcher-title-highlight">{t('launcherLink.title')}</span>
          </Title>
          <Paragraph className="launcher-intro">{t('launcherLink.intro')}</Paragraph>
        </div>

        <div className="launcher-hero-visual" aria-hidden>
          <div className="launcher-orbit">
            <div className="launcher-orbit-ring launcher-orbit-ring--outer" />
            <div className="launcher-orbit-ring launcher-orbit-ring--inner" />
            <div className="launcher-orbit-core">{icon}</div>
          </div>
        </div>
      </div>
    </section>
  );
}

export function LauncherLinkPage() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const deviceId = params.get('device')?.trim() ?? '';
  const { isAuthenticated, user } = useAuth();
  const { openAuthModal } = useAuthModal();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [linked, setLinked] = useState(false);
  const [deviceInfo, setDeviceInfo] = useState<DeviceStatus | null>(null);
  const [deviceLoading, setDeviceLoading] = useState(false);

  useEffect(() => {
    if (!deviceId) {
      setError(t('launcherLink.missingDevice'));
      return;
    }

    let cancelled = false;
    setDeviceLoading(true);
    setError(null);

    void (async () => {
      try {
        const status = await api.deviceStatus(deviceId);
        if (cancelled) return;
        setDeviceInfo(status);
        if (status.status === 'linked') {
          setError(t('launcherLink.alreadyLinked'));
        } else if (status.status === 'expired') {
          setError(t('launcherLink.linkExpired'));
        }
      } catch (e) {
        if (cancelled) return;
        if (e instanceof Error && !isBackendUnavailableError(e)) {
          setError(e.message);
        } else {
          setError(t('launcherLink.deviceNotFound'));
        }
      } finally {
        if (!cancelled) {
          setDeviceLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [deviceId, t]);

  const canLink = deviceInfo?.status === 'pending_link' && !linked;

  const handleLink = async () => {
    if (!canLink || !isAuthenticated) return;
    setLoading(true);
    setError(null);
    try {
      const result = await api.linkDevice({ device_id: deviceId });
      setLinked(true);
      setDeviceInfo((prev) => (prev ? { ...prev, status: 'linked' } : prev));
      saveLinkedDevice(deviceId);
      logger.info('device linked', { deviceId, ownerType: result.owner_type });
    } catch (e) {
      if (e instanceof Error && !isBackendUnavailableError(e)) {
        setError(e.message);
      } else {
        setError(t('launcherLink.linkFailed'));
      }
    } finally {
      setLoading(false);
    }
  };

  if (!deviceId) {
    return (
      <div className="launcher-page launcher-link-page">
        <LinkHero variant="error" icon={<ExclamationCircleOutlined />} />
        <section className="launcher-section launcher-section--workspace">
          <Alert
            type="error"
            title={t('launcherLink.linkError')}
            description={error ?? t('launcherLink.invalidLink')}
            showIcon
          />
          <div className="launcher-link-error-actions">
            <Link to="/launcher">
              <Button type="primary">{t('launcherLink.toLauncher')}</Button>
            </Link>
          </div>
        </section>
      </div>
    );
  }

  if (linked) {
    return (
      <div className="launcher-page launcher-link-page">
        <LinkHero variant="success" icon={<CheckCircleOutlined />} />
        <section className="launcher-section launcher-section--workspace">
          <DeviceInfoPanel deviceId={deviceId} info={deviceInfo} loading={false} />
          <div className="launcher-status-card launcher-status-card--linked">
            <span className="launcher-status-icon">
              <CheckCircleOutlined />
            </span>
            <div className="launcher-status-content">
              <Text strong className="launcher-status-title">
                {t('launcherLink.deviceLinked')}
              </Text>
              <Paragraph className="launcher-status-desc">
                {t('launcherLink.linkedAsUser')}
              </Paragraph>
              <div className="launcher-link-success-actions">
                <Link to="/launcher">
                  <Button type="primary" size="large">
                    {t('launcherLink.goToInstances')}
                  </Button>
                </Link>
              </div>
            </div>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="launcher-page launcher-link-page">
      <LinkHero icon={<LinkOutlined />} />

      <section className="launcher-section launcher-section--workspace">
        <DeviceInfoPanel deviceId={deviceId} info={deviceInfo} loading={deviceLoading} />

        <p className="launcher-link-security">{t('launcherLink.securityNote')}</p>

        {error && (
          <Alert
            type={deviceInfo?.status === 'linked' ? 'info' : 'error'}
            title={error}
            description={
              deviceInfo?.status === 'linked'
                ? t('launcherLink.alreadyLinkedHint')
                : deviceInfo?.status === 'expired'
                  ? t('launcherLink.linkExpiredHint')
                  : undefined
            }
            showIcon
            style={{ marginBottom: 24 }}
            action={
              deviceInfo?.status === 'linked' ? (
                <Link to="/launcher">
                  <Button size="small">{t('launcherLink.goToInstances')}</Button>
                </Link>
              ) : undefined
            }
          />
        )}

        <div className="launcher-link-options launcher-link-options--single">
          {isAuthenticated ? (
            <article className="launcher-link-option launcher-link-option--recommended">
              <span className="launcher-link-option-badge">{t('launcherLink.accountRecommended')}</span>
              <div className="launcher-link-option-head">
                <span className="launcher-link-option-icon">
                  <UserOutlined />
                </span>
                <Title level={4} className="launcher-link-option-title">
                  {t('launcherLink.linkToAccount')}
                </Title>
              </div>
              <Paragraph className="launcher-link-option-desc">
                {user?.email
                  ? t('launcherLink.signedInAs', { email: user.email })
                  : t('launcherLink.linkToAccountHint')}
              </Paragraph>
              <Paragraph className="launcher-link-option-desc">{t('launcherLink.accountBenefits')}</Paragraph>
              <div className="launcher-link-option-actions">
                <Button
                  type="primary"
                  size="large"
                  block
                  icon={<LinkOutlined />}
                  loading={loading}
                  disabled={!canLink || deviceLoading}
                  onClick={() => void handleLink()}
                >
                  {t('launcherLink.linkDevice')}
                </Button>
              </div>
            </article>
          ) : (
            <article className="launcher-link-option launcher-link-option--recommended">
              <div className="launcher-link-option-head">
                <span className="launcher-link-option-icon">
                  <LoginOutlined />
                </span>
                <Title level={4} className="launcher-link-option-title">
                  {t('launcherLink.signInRequiredTitle')}
                </Title>
              </div>
              <Paragraph className="launcher-link-option-desc">{t('launcherLink.signInRequiredDesc')}</Paragraph>
              <div className="launcher-link-option-actions">
                <Button type="primary" size="large" block onClick={() => openAuthModal('login')}>
                  {t('auth.signIn')}
                </Button>
                <Button size="large" block onClick={() => openAuthModal('register')}>
                  {t('auth.createAccount')}
                </Button>
              </div>
            </article>
          )}
        </div>
      </section>
    </div>
  );
}
