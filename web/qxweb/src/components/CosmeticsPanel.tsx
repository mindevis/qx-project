import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Card, Popconfirm, Segmented, Space, Spin, Tag, Typography, Upload } from 'antd';
import { CheckCircleFilled, DeleteOutlined, LinkOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadProps } from 'antd';
import { SkinViewer } from 'skinview3d';
import { api, type MojangLinkStatus, type ProfileModel, type UserCosmetics } from '@/api/client';
import { ProfileModelPicker } from '@/components/ProfileModelPicker';
import { useMessage } from '@/hooks/useMessage';
import { useI18n } from '@/i18n/I18nContext';
import { logger } from '@/lib/logger';
import { officialAccountBodyUrl, officialAccountSkinUrl } from '@/lib/mojangSkin';
import './CosmeticsPanel.css';

const { Paragraph, Text } = Typography;

const MODEL_TYPE = {
  steve: 'default',
  alex: 'slim',
} as const;

const DEFAULT_SKIN: Record<ProfileModel, string> = {
  steve: '/profiles/steve-skin.png',
  alex: '/profiles/alex-skin.png',
};

const PREVIEW_WIDTH = 180;
const PREVIEW_HEIGHT = 240;

type CapeType = 'none' | 'qx' | 'custom';
type SkinSource = 'custom' | 'official' | 'default';

type CosmeticsPreviewProps = {
  model: ProfileModel;
  skinUrl?: string;
  capeUrl?: string;
  fallbackBodyUrl?: string;
};

function CosmeticsPreview({ model, skinUrl, capeUrl, fallbackBodyUrl }: CosmeticsPreviewProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const viewerRef = useRef<SkinViewer | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [skinUrl, model, fallbackBodyUrl]);

  useEffect(() => {
    if (failed) return;
    const canvas = canvasRef.current;
    if (!canvas) return;

    let viewer: SkinViewer;
    try {
      viewer = new SkinViewer({
        canvas,
        width: PREVIEW_WIDTH,
        height: PREVIEW_HEIGHT,
        skin: skinUrl ?? DEFAULT_SKIN[model],
        model: MODEL_TYPE[model],
      });
    } catch (error) {
      logger.warn('cosmetics preview unavailable', { error: String(error) });
      setFailed(true);
      return;
    }
    viewer.background = null;
    viewer.autoRotate = true;
    viewer.controls.enableZoom = false;
    viewer.controls.enablePan = false;
    viewerRef.current = viewer;
    return () => {
      viewer.dispose();
      viewerRef.current = null;
    };
  }, [failed, model, skinUrl]);

  useEffect(() => {
    const viewer = viewerRef.current;
    if (!viewer || viewer.disposed || failed) return;
    const src = skinUrl ?? DEFAULT_SKIN[model];
    void viewer.loadSkin(src, { model: MODEL_TYPE[model] }).then(
      () => {
        if (!viewer.disposed) viewer.resetCameraPose();
      },
      (error: unknown) => {
        logger.warn('cosmetics skin load failed', { error: String(error) });
        setFailed(true);
      },
    );
  }, [failed, model, skinUrl]);

  useEffect(() => {
    const viewer = viewerRef.current;
    if (!viewer || viewer.disposed || failed) return;
    if (capeUrl) {
      void viewer.loadCape(capeUrl);
    } else {
      viewer.loadCape(null);
    }
  }, [capeUrl, failed]);

  if (failed && fallbackBodyUrl) {
    return (
      <img
        src={fallbackBodyUrl}
        alt=""
        className="cosmetics-preview-body"
        width={PREVIEW_WIDTH}
        height={PREVIEW_HEIGHT}
      />
    );
  }
  if (failed) return null;
  return (
    <canvas
      ref={canvasRef}
      className="cosmetics-preview-canvas"
      width={PREVIEW_WIDTH}
      height={PREVIEW_HEIGHT}
      aria-hidden
    />
  );
}

type CosmeticsPanelProps = {
  embedded?: boolean;
};

export function CosmeticsPanel({ embedded = false }: CosmeticsPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [cosmetics, setCosmetics] = useState<UserCosmetics | null>(null);
  const [mojang, setMojang] = useState<MojangLinkStatus | null>(null);
  const [model, setModel] = useState<ProfileModel>('steve');
  const [capeType, setCapeType] = useState<CapeType>('none');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [data, status] = await Promise.all([
        api.getCosmetics(),
        api.mojangStatus().catch(() => null),
      ]);
      setCosmetics(data);
      setMojang(status);
      setModel(data.skin_model ?? 'steve');
      setCapeType((data.cape_type as CapeType) ?? (data.has_cape ? 'custom' : 'none'));
    } catch (e) {
      logger.warn('failed to load cosmetics', { error: String(e) });
      message.error(t('cosmetics.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const saveEquip = async (patch: { skin_model?: ProfileModel; cape_type?: CapeType }) => {
    setSaving(true);
    try {
      const data = await api.updateCosmetics(patch);
      setCosmetics(data);
      if (patch.cape_type) setCapeType(patch.cape_type);
      message.success(t('cosmetics.saved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('cosmetics.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const skinUploadProps: UploadProps = {
    accept: 'image/png',
    showUploadList: false,
    beforeUpload: (file) => {
      void (async () => {
        setSaving(true);
        try {
          const data = await api.uploadCosmeticsSkin(file);
          setCosmetics(data);
          message.success(t('cosmetics.skinUploaded'));
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('cosmetics.uploadFailed'));
        } finally {
          setSaving(false);
        }
      })();
      return false;
    },
  };

  const capeUploadProps: UploadProps = {
    accept: 'image/png',
    showUploadList: false,
    beforeUpload: (file) => {
      void (async () => {
        setSaving(true);
        try {
          const data = await api.uploadCosmeticsCape(file);
          setCosmetics(data);
          setCapeType('custom');
          message.success(t('cosmetics.capeUploaded'));
        } catch (e) {
          message.error(e instanceof Error ? e.message : t('cosmetics.uploadFailed'));
        } finally {
          setSaving(false);
        }
      })();
      return false;
    },
  };

  const cacheBust = cosmetics?.updated_at ? `?v=${encodeURIComponent(cosmetics.updated_at)}` : '';
  const officialSkinUrl = officialAccountSkinUrl(mojang?.minecraft_uuid, mojang?.username);
  const officialBodyUrl = officialAccountBodyUrl(mojang?.minecraft_uuid, mojang?.username);
  const customSkinUrl =
    cosmetics?.has_skin && cosmetics.skin_url ? `${cosmetics.skin_url}${cacheBust}` : undefined;
  const previewSkinUrl = customSkinUrl ?? officialSkinUrl;
  const previewCapeUrl =
    cosmetics?.has_cape && cosmetics.cape_url ? `${cosmetics.cape_url}${cacheBust}` : undefined;
  const skinSource: SkinSource = cosmetics?.has_skin
    ? 'custom'
    : mojang?.linked && officialSkinUrl
      ? 'official'
      : 'default';
  const statusLabel =
    skinSource === 'custom'
      ? t('cosmetics.skinApplied')
      : skinSource === 'official'
        ? t('cosmetics.skinOfficial')
        : t('cosmetics.skinDefault');

  const importOfficialSkin = async () => {
    const username = mojang?.username?.trim();
    if (!username) return;
    setSaving(true);
    try {
      const data = await api.applyCosmeticsSkin({ username });
      setCosmetics(data);
      if (data.skin_model) setModel(data.skin_model);
      message.success(t('cosmetics.skinImported'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('cosmetics.importFailed'));
    } finally {
      setSaving(false);
    }
  };

  const content = loading ? (
    <Spin />
  ) : (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      {!embedded ? <p className="cosmetics-hint">{t('cosmetics.hint')}</p> : null}
      <div className="cosmetics-preview-row">
        <div
          className={`cosmetics-preview${skinSource !== 'default' ? ' cosmetics-preview--applied' : ''}`}
        >
          <CosmeticsPreview
            model={model}
            skinUrl={previewSkinUrl}
            capeUrl={previewCapeUrl}
            fallbackBodyUrl={officialBodyUrl}
          />
          <Tag
            className="cosmetics-preview-status"
            color={skinSource === 'custom' ? 'success' : skinSource === 'official' ? 'processing' : 'default'}
            icon={skinSource !== 'default' ? <CheckCircleFilled /> : undefined}
          >
            {statusLabel}
          </Tag>
          {mojang?.linked && mojang.username ? (
            <Text type="secondary" className="cosmetics-preview-account">
              {mojang.username}
            </Text>
          ) : null}
        </div>
        <div className="cosmetics-controls">
          {skinSource === 'official' ? (
            <Paragraph type="secondary" className="cosmetics-official-hint">
              {t('cosmetics.skinOfficialHint')}
            </Paragraph>
          ) : null}
          {!mojang?.linked ? (
            <Paragraph type="secondary" className="cosmetics-official-hint">
              {t('cosmetics.mojangNotLinked')}{' '}
              <Typography.Link href="/profile">
                <LinkOutlined /> {t('cosmetics.goToProfile')}
              </Typography.Link>
            </Paragraph>
          ) : null}
          <div className="cosmetics-field">
            <span className="cosmetics-label">{t('cosmetics.skinModel')}</span>
            <ProfileModelPicker
              value={model}
              onChange={(value) => {
                setModel(value);
                void saveEquip({ skin_model: value });
              }}
            />
          </div>
          <Space wrap>
            <Upload {...skinUploadProps}>
              <Button icon={<UploadOutlined />} loading={saving}>
                {t('cosmetics.uploadSkin')}
              </Button>
            </Upload>
            {skinSource === 'official' && mojang?.username ? (
              <Button loading={saving} onClick={() => void importOfficialSkin()}>
                {t('cosmetics.importOfficialSkin')}
              </Button>
            ) : null}
            {cosmetics?.has_skin ? (
              <Popconfirm
                title={
                  mojang?.linked ? t('cosmetics.resetSkinConfirmOfficial') : t('cosmetics.resetSkinConfirm')
                }
                okText={t('cosmetics.resetSkin')}
                cancelText={t('common.cancel')}
                okButtonProps={{ danger: true }}
                onConfirm={() => void api.deleteCosmeticsSkin().then(load)}
              >
                <Button danger icon={<DeleteOutlined />} loading={saving}>
                  {t('cosmetics.resetSkin')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
          <div className="cosmetics-field">
            <span className="cosmetics-label">{t('cosmetics.cape')}</span>
            <Segmented<CapeType>
              value={capeType}
              onChange={(value) => {
                setCapeType(value);
                void saveEquip({ cape_type: value });
              }}
              options={[
                { value: 'none', label: t('cosmetics.capeNone') },
                { value: 'qx', label: t('cosmetics.capeQX') },
                { value: 'custom', label: t('cosmetics.capeCustom') },
              ]}
            />
          </div>
          {capeType === 'custom' ? (
            <Space wrap>
              <Upload {...capeUploadProps}>
                <Button icon={<UploadOutlined />} loading={saving}>
                  {t('cosmetics.uploadCape')}
                </Button>
              </Upload>
              {cosmetics?.has_cape && cosmetics.cape_type === 'custom' ? (
                <Popconfirm
                  title={t('cosmetics.resetCapeConfirm')}
                  okText={t('cosmetics.resetCape')}
                  cancelText={t('common.cancel')}
                  okButtonProps={{ danger: true }}
                  onConfirm={() => void api.deleteCosmeticsCape().then(load)}
                >
                  <Button danger icon={<DeleteOutlined />} loading={saving}>
                    {t('cosmetics.resetCape')}
                  </Button>
                </Popconfirm>
              ) : null}
            </Space>
          ) : null}
        </div>
      </div>
      <Typography.Paragraph type="secondary" className="cosmetics-note">
        {t('cosmetics.skinServerNote')}
      </Typography.Paragraph>
    </Space>
  );

  if (embedded) {
    return <div className="cosmetics-panel cosmetics-panel--embedded">{content}</div>;
  }

  return (
    <Card title={t('cosmetics.title')} style={{ maxWidth: 720 }} className="cosmetics-panel">
      {content}
    </Card>
  );
}
