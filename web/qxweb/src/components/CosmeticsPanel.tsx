import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Card, Radio, Space, Spin, Upload } from 'antd';
import { DeleteOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadProps } from 'antd';
import { SkinViewer } from 'skinview3d';
import { api, type ProfileModel, type UserCosmetics } from '@/api/client';
import { ProfileModelPicker } from '@/components/ProfileModelPicker';
import { useMessage } from '@/hooks/useMessage';
import { useI18n } from '@/i18n/I18nContext';
import { logger } from '@/lib/logger';
import './CosmeticsPanel.css';

const MODEL_TYPE = {
  steve: 'default',
  alex: 'slim',
} as const;

const DEFAULT_SKIN: Record<ProfileModel, string> = {
  steve: '/profiles/steve-skin.png',
  alex: '/profiles/alex-skin.png',
};

type CapeType = 'none' | 'qx' | 'custom';

type CosmeticsPreviewProps = {
  model: ProfileModel;
  skinUrl?: string;
  capeUrl?: string;
};

function CosmeticsPreview({ model, skinUrl, capeUrl }: CosmeticsPreviewProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const viewerRef = useRef<SkinViewer | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (failed) return;
    const canvas = canvasRef.current;
    if (!canvas) return;

    let viewer: SkinViewer;
    try {
      viewer = new SkinViewer({
        canvas,
        width: 140,
        height: 180,
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
    void viewer.loadSkin(src, { model: MODEL_TYPE[model] }).then(() => {
      if (!viewer.disposed) viewer.resetCameraPose();
    });
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

  if (failed) return null;
  return <canvas ref={canvasRef} className="cosmetics-preview-canvas" width={140} height={180} aria-hidden />;
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
  const [model, setModel] = useState<ProfileModel>('steve');
  const [capeType, setCapeType] = useState<CapeType>('none');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getCosmetics();
      setCosmetics(data);
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
  const previewSkinUrl = cosmetics?.has_skin && cosmetics.skin_url
    ? `${cosmetics.skin_url}${cacheBust}`
    : undefined;
  const previewCapeUrl = cosmetics?.has_cape && cosmetics.cape_url
    ? `${cosmetics.cape_url}${cacheBust}`
    : undefined;

  const content = loading ? (
    <Spin />
  ) : (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      {!embedded ? <p className="cosmetics-hint">{t('cosmetics.hint')}</p> : null}
      <div className="cosmetics-preview-row">
        <CosmeticsPreview model={model} skinUrl={previewSkinUrl} capeUrl={previewCapeUrl} />
        <div className="cosmetics-controls">
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
            {cosmetics?.has_skin ? (
              <Button
                danger
                icon={<DeleteOutlined />}
                loading={saving}
                onClick={() => void api.deleteCosmeticsSkin().then(load)}
              >
                {t('cosmetics.resetSkin')}
              </Button>
            ) : null}
          </Space>
          <div className="cosmetics-field">
            <span className="cosmetics-label">{t('cosmetics.cape')}</span>
            <Radio.Group
              value={capeType}
              onChange={(e) => {
                const value = e.target.value as CapeType;
                setCapeType(value);
                void saveEquip({ cape_type: value });
              }}
            >
              <Radio.Button value="none">{t('cosmetics.capeNone')}</Radio.Button>
              <Radio.Button value="qx">{t('cosmetics.capeQX')}</Radio.Button>
              <Radio.Button value="custom">{t('cosmetics.capeCustom')}</Radio.Button>
            </Radio.Group>
          </div>
          {capeType === 'custom' ? (
            <Space wrap>
              <Upload {...capeUploadProps}>
                <Button icon={<UploadOutlined />} loading={saving}>
                  {t('cosmetics.uploadCape')}
                </Button>
              </Upload>
              {cosmetics?.has_cape && cosmetics.cape_type === 'custom' ? (
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  loading={saving}
                  onClick={() => void api.deleteCosmeticsCape().then(load)}
                >
                  {t('cosmetics.resetCape')}
                </Button>
              ) : null}
            </Space>
          ) : null}
        </div>
      </div>
      <p className="cosmetics-note">{t('cosmetics.skinServerNote')}</p>
    </Space>
  );

  if (embedded) {
    return <div className="cosmetics-panel cosmetics-panel--embedded">{content}</div>;
  }

  return (
    <Card title={t('cosmetics.title')} style={{ maxWidth: 560 }} className="cosmetics-panel">
      {content}
    </Card>
  );
}
