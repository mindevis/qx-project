import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Form, Select, Typography } from 'antd';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { fallbackMcVersionsList } from '@/launcher/mcVersions';
import { cachedListMcVersions } from '@/lib/mcVersionsCache';
import { logger } from '@/lib/logger';
import {
  changeVpsGameServerVersion,
  type VpsGameServer,
} from '@/lib/vpsGameServers';
import {
  gameServerTypeNeedsLoader,
  listGameServerLoaderVersions,
  listGameServerMcVersions,
  type VersionOption,
} from '@/lib/gameServerVersions';
import type { VpsGameServerType } from '@/lib/gameServerTypes';

const { Paragraph, Title } = Typography;

type VersionFormValues = {
  mc_version: string;
  loader_version?: string;
};

type GameServerVersionSettingsPanelProps = {
  vpsId: string;
  game: VpsGameServer;
  disabled?: boolean;
  onUpdated: (game: VpsGameServer) => void;
};

function ensureOption(options: VersionOption[], value?: string): VersionOption[] {
  if (!value) return options;
  if (options.some((option) => option.value === value)) return options;
  return [{ value, label: value }, ...options];
}

function coreVersionLabel(
  t: (key: string, params?: Record<string, string | number>) => string,
  serverType: VpsGameServerType,
): string {
  if (serverType === 'vanilla') {
    return t('servers.gameServerCoreVersion');
  }
  return t('servers.gameServerCoreVersionWithType', {
    type: t(`servers.gameServerType.${serverType}`),
  });
}

export function GameServerVersionSettingsPanel({
  vpsId,
  game,
  disabled,
  onUpdated,
}: GameServerVersionSettingsPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const [form] = Form.useForm<VersionFormValues>();
  const [saving, setSaving] = useState(false);
  const [mcOptions, setMcOptions] = useState<VersionOption[]>([]);
  const [loaderOptions, setLoaderOptions] = useState<VersionOption[]>([]);
  const [mcLoading, setMcLoading] = useState(false);
  const [loaderLoading, setLoaderLoading] = useState(false);
  const loadSeqRef = useRef(0);
  const serverType = (game.server_type ?? 'vanilla') as VpsGameServerType;
  const needsLoader = gameServerTypeNeedsLoader(serverType);
  const mcVersion = Form.useWatch('mc_version', form);
  const loaderVersion = Form.useWatch('loader_version', form);
  const unchanged =
    (mcVersion ?? '') === (game.mc_version ?? '') &&
    (needsLoader ? (loaderVersion ?? '') === (game.loader_version ?? '') : true);

  const loadLoaderOptions = useCallback(
    async (mc: string, seq: number, preferred?: string) => {
      if (!needsLoader || !mc) {
        setLoaderOptions([]);
        form.setFieldValue('loader_version', undefined);
        return;
      }
      setLoaderLoading(true);
      try {
        const options = ensureOption(await listGameServerLoaderVersions(serverType, mc), preferred);
        if (seq !== loadSeqRef.current) return;
        setLoaderOptions(options);
        const current = form.getFieldValue('loader_version') as string | undefined;
        const next =
          (current && options.some((option) => option.value === current) && current) ||
          (preferred && options.some((option) => option.value === preferred) && preferred) ||
          options[0]?.value;
        form.setFieldValue('loader_version', next);
      } catch (e) {
        if (seq !== loadSeqRef.current) return;
        logger.warn('failed to load loader versions', { serverType, mc, error: String(e) });
        message.warning(t('servers.gameServerLoaderVersionsLoadFailed'));
        setLoaderOptions(ensureOption([], preferred));
        form.setFieldValue('loader_version', preferred);
      } finally {
        if (seq === loadSeqRef.current) {
          setLoaderLoading(false);
        }
      }
    },
    [form, message, needsLoader, serverType, t],
  );

  useEffect(() => {
    const seq = ++loadSeqRef.current;
    let cancelled = false;
    form.setFieldsValue({
      mc_version: game.mc_version,
      loader_version: game.loader_version,
    });
    const load = async () => {
      setMcLoading(true);
      try {
        let fallback = fallbackMcVersionsList().items;
        try {
          const result = await cachedListMcVersions();
          fallback = result.items ?? fallback;
        } catch (e) {
          logger.warn('failed to load mc versions', { error: String(e) });
        }
        const options = ensureOption(
          await listGameServerMcVersions(serverType, fallback),
          game.mc_version,
        );
        if (cancelled || seq !== loadSeqRef.current) return;
        setMcOptions(options);
        const nextMc =
          (game.mc_version && options.some((option) => option.value === game.mc_version)
            ? game.mc_version
            : options[0]?.value) ?? '';
        form.setFieldValue('mc_version', nextMc);
        await loadLoaderOptions(nextMc, seq, game.loader_version);
      } catch (e) {
        if (cancelled || seq !== loadSeqRef.current) return;
        logger.warn('failed to load mc versions for server type', {
          serverType,
          error: String(e),
        });
        message.warning(t('servers.gameServerMcVersionsLoadFailed'));
        setMcOptions(ensureOption([], game.mc_version));
      } finally {
        if (seq === loadSeqRef.current) {
          setMcLoading(false);
        }
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [form, game.id, game.loader_version, game.mc_version, loadLoaderOptions, message, serverType, t]);

  const onMcChange = (value: string) => {
    const seq = ++loadSeqRef.current;
    form.setFieldValue('loader_version', undefined);
    void loadLoaderOptions(value, seq);
  };

  const apply = async () => {
    let values: VersionFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    if (unchanged) {
      message.info(t('gameServerDetail.versionUnchanged'));
      return;
    }
    setSaving(true);
    try {
      const updated = await changeVpsGameServerVersion(vpsId, game.id, {
        mc_version: values.mc_version,
        loader_version: needsLoader ? values.loader_version : undefined,
      });
      onUpdated(updated);
      message.success(t('gameServerDetail.versionStarted'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.versionFailed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="game-server-version-settings">
      <Title level={4} className="game-server-launch-title">
        {t('gameServerDetail.versionTitle')}
      </Title>
      <Paragraph type="secondary">{t('gameServerDetail.versionHint')}</Paragraph>
      <Form form={form} layout="vertical" disabled={disabled}>
        <Form.Item
          name="mc_version"
          label={t('servers.gameServerMcVersion')}
          rules={[{ required: true, message: t('servers.gameServerMcVersionRequired') }]}
        >
          <Select
            showSearch
            optionFilterProp="label"
            options={mcOptions}
            loading={mcLoading}
            placeholder={t('servers.gameServerMcVersionRequired')}
            onChange={onMcChange}
          />
        </Form.Item>
        {needsLoader ? (
          <Form.Item
            name="loader_version"
            label={coreVersionLabel(t, serverType)}
            rules={[{ required: true, message: t('servers.gameServerCoreVersionRequired') }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              options={loaderOptions}
              loading={loaderLoading}
              placeholder={t('servers.gameServerCoreVersionRequired')}
            />
          </Form.Item>
        ) : null}
        <Button
          type="primary"
          loading={saving}
          disabled={disabled || unchanged || mcLoading || loaderLoading}
          onClick={() => void apply()}
        >
          {t('gameServerDetail.versionApply')}
        </Button>
      </Form>
    </section>
  );
}
