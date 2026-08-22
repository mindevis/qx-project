import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Checkbox,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Typography,
} from 'antd';
import { GlobalOutlined, PlusOutlined, RocketOutlined } from '@ant-design/icons';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { modalMotionProps } from '@/lib/modal';
import {
  DEFAULT_MC_VERSION,
  fallbackMcVersionsList,
  pickDefaultMcVersion,
  type McVersionItem,
} from '@/launcher/mcVersions';
import { logger } from '@/lib/logger';
import { cachedListMcVersions } from '@/lib/mcVersionsCache';
import {
  addVpsGameServer,
  suggestDefaultGamePort,
  type VpsGameServer,
} from '@/lib/vpsGameServers';
import {
  DEFAULT_GAME_SERVER_TYPE,
  GAME_SERVER_TYPE_GROUPS,
  gameServerTypeCapabilities,
  gameServerPrimaryVersionI18nKey,
  gameServerPrimaryVersionRequiredI18nKey,
  type VpsGameServerType,
} from '@/lib/gameServerTypes';
import {
  gameServerTypeNeedsLoader,
  listGameServerLoaderVersions,
  listGameServerMcVersions,
  type VersionOption,
} from '@/lib/gameServerVersions';
import '@/pages/ServersPage.css';

const { Paragraph, Text } = Typography;

function useGameServerTypeSelectOptions() {
  const { t } = useI18n();
  return useMemo(
    () =>
      GAME_SERVER_TYPE_GROUPS.map((group) => ({
        label: t(`servers.gameServerTypeGroup.${group.id}`),
        options: group.types.map((type) => ({
          value: type,
          label: t(`servers.gameServerType.${type}`),
        })),
      })),
    [t],
  );
}

type GameServerTypeFieldProps = {
  value?: VpsGameServerType;
  onChange?: (value: VpsGameServerType) => void;
};

function GameServerTypeField({ value, onChange }: GameServerTypeFieldProps) {
  const { t } = useI18n();
  const options = useGameServerTypeSelectOptions();

  return (
    <Select
      showSearch
      optionFilterProp="label"
      value={value}
      options={options}
      onChange={onChange}
      placeholder={t('servers.gameServerTypeRequired')}
    />
  );
}

type McVersionSelectProps = {
  value?: string;
  onChange?: (value: string) => void;
  options: VersionOption[];
  loading?: boolean;
  placeholder: string;
  onMcVersionChange?: (value: string) => void;
};

function McVersionSelect({
  value,
  onChange,
  options,
  loading,
  placeholder,
  onMcVersionChange,
}: McVersionSelectProps) {
  return (
    <Select
      showSearch
      optionFilterProp="label"
      value={value}
      options={options}
      loading={loading}
      placeholder={placeholder}
      onChange={(next) => {
        onChange?.(next);
        onMcVersionChange?.(next);
      }}
    />
  );
}

type LoaderVersionSelectProps = {
  value?: string;
  onChange?: (value: string) => void;
  options: VersionOption[];
  loading?: boolean;
  disabled?: boolean;
  placeholder: string;
};

function LoaderVersionSelect({
  value,
  onChange,
  options,
  loading,
  disabled,
  placeholder,
}: LoaderVersionSelectProps) {
  return (
    <Select
      showSearch
      optionFilterProp="label"
      value={value}
      options={options}
      loading={loading}
      disabled={disabled}
      placeholder={placeholder}
      onChange={onChange}
    />
  );
}

function gameServerCoreVersionLabel(
  t: (key: string, params?: Record<string, string | number>) => string,
  serverType: VpsGameServerType | undefined,
): string {
  if (!serverType || serverType === 'vanilla') {
    return t('servers.gameServerCoreVersion');
  }
  return t('servers.gameServerCoreVersionWithType', {
    type: t(`servers.gameServerType.${serverType}`),
  });
}

export function AddGameServerModal({
  open,
  vpsId,
  defaultAddress,
  existingGames,
  onClose,
  onCreated,
}: {
  open: boolean;
  vpsId: string;
  defaultAddress: string;
  existingGames: VpsGameServer[];
  onClose: () => void;
  onCreated: () => void;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [mcVersionsLoading, setMcVersionsLoading] = useState(false);
  const [mcVersionOptions, setMcVersionOptions] = useState<VersionOption[]>([]);
  const [loaderVersionOptions, setLoaderVersionOptions] = useState<VersionOption[]>([]);
  const [mcOptionsLoading, setMcOptionsLoading] = useState(false);
  const [loaderOptionsLoading, setLoaderOptionsLoading] = useState(false);
  const serverType = Form.useWatch('server_type', form) as VpsGameServerType | undefined;
  const showInMonitoring = Form.useWatch('show_in_monitoring', form) as boolean | undefined;
  const mcVersionsRef = useRef<McVersionItem[]>([]);
  const defaultMcVersionRef = useRef(DEFAULT_MC_VERSION);
  const versionsLoadSeqRef = useRef(0);
  const wasOpenRef = useRef(false);
  const needsLoader = serverType ? gameServerTypeNeedsLoader(serverType) : false;

  const pickMcVersion = useCallback(
    (type: VpsGameServerType, options: VersionOption[], current?: string) => {
      if (current && options.some((option) => option.value === current)) {
        return current;
      }
      if (type === 'vanilla') {
        return (
          options.find((option) => option.value === defaultMcVersionRef.current)?.value ??
          options[0]?.value
        );
      }
      return options[0]?.value;
    },
    [],
  );

  const patchVersionFields = useCallback(
    (patch: { mc_version?: string; loader_version?: string }) => {
      const next: { mc_version?: string; loader_version?: string } = {};
      if (
        patch.mc_version !== undefined &&
        form.getFieldValue('mc_version') !== patch.mc_version
      ) {
        next.mc_version = patch.mc_version;
      }
      if (
        patch.loader_version !== undefined &&
        form.getFieldValue('loader_version') !== patch.loader_version
      ) {
        next.loader_version = patch.loader_version;
      }
      if (Object.keys(next).length > 0) {
        form.setFieldsValue(next);
      }
    },
    [form],
  );

  const loadLoaderVersionOptions = useCallback(
    async (type: VpsGameServerType, mcVersion: string, seq: number) => {
      if (!gameServerTypeNeedsLoader(type) || !mcVersion) {
        setLoaderVersionOptions([]);
        patchVersionFields({ loader_version: '' });
        return;
      }
      setLoaderOptionsLoading(true);
      try {
        const options = await listGameServerLoaderVersions(type, mcVersion);
        if (seq !== versionsLoadSeqRef.current) return;
        setLoaderVersionOptions(options);
        const current = form.getFieldValue('loader_version') as string | undefined;
        const nextLoader =
          current && options.some((option) => option.value === current)
            ? current
            : options[0]?.value ?? '';
        patchVersionFields({ loader_version: nextLoader });
      } catch (e) {
        if (seq !== versionsLoadSeqRef.current) return;
        logger.warn('failed to load loader versions', { type, mcVersion, error: String(e) });
        message.warning(t('servers.gameServerLoaderVersionsLoadFailed'));
        setLoaderVersionOptions([]);
        patchVersionFields({ loader_version: '' });
      } finally {
        if (seq === versionsLoadSeqRef.current) {
          setLoaderOptionsLoading(false);
        }
      }
    },
    [form, message, patchVersionFields, t],
  );

  const refreshVersionsForType = useCallback(
    async (type: VpsGameServerType, resetMc = false) => {
      const seq = ++versionsLoadSeqRef.current;
      setMcOptionsLoading(true);
      setLoaderVersionOptions([]);
      try {
        const options = await listGameServerMcVersions(type, mcVersionsRef.current);
        if (seq !== versionsLoadSeqRef.current) return;
        setMcVersionOptions(options);
        const current = resetMc ? undefined : (form.getFieldValue('mc_version') as string | undefined);
        const nextMc = pickMcVersion(type, options, current) ?? '';
        patchVersionFields({
          mc_version: nextMc,
          loader_version: '',
        });
        if (nextMc && gameServerTypeNeedsLoader(type)) {
          await loadLoaderVersionOptions(type, nextMc, seq);
        }
      } catch (e) {
        if (seq !== versionsLoadSeqRef.current) return;
        logger.warn('failed to load mc versions for server type', { type, error: String(e) });
        message.warning(t('servers.gameServerMcVersionsLoadFailed'));
      } finally {
        if (seq === versionsLoadSeqRef.current) {
          setMcOptionsLoading(false);
        }
      }
    },
    [form, loadLoaderVersionOptions, message, patchVersionFields, pickMcVersion, t],
  );

  const handleMcVersionChange = useCallback(
    (mcVersion: string) => {
      const type =
        (form.getFieldValue('server_type') as VpsGameServerType | undefined) ??
        DEFAULT_GAME_SERVER_TYPE;
      if (!gameServerTypeNeedsLoader(type)) return;
      const seq = ++versionsLoadSeqRef.current;
      patchVersionFields({ loader_version: '' });
      void loadLoaderVersionOptions(type, mcVersion, seq);
    },
    [form, loadLoaderVersionOptions, patchVersionFields],
  );

  const loadMcVersions = useCallback(async () => {
    setMcVersionsLoading(true);
    try {
      const result = await cachedListMcVersions();
      const items = result.items ?? [];
      mcVersionsRef.current = items;
      const nextDefault = pickDefaultMcVersion(result.latest, items);
      defaultMcVersionRef.current = nextDefault;
    } catch (e) {
      logger.warn('failed to load mc versions', { error: String(e) });
      message.warning(t('launcher.mcVersionsLoadFailed'));
      const fallback = fallbackMcVersionsList();
      mcVersionsRef.current = fallback.items;
      const nextDefault = pickDefaultMcVersion(undefined, fallback.items);
      defaultMcVersionRef.current = nextDefault;
    } finally {
      setMcVersionsLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    if (open) {
      void loadMcVersions();
    }
  }, [open, loadMcVersions]);

  useEffect(() => {
    if (!open || mcVersionsLoading) return;
    const type = serverType ?? DEFAULT_GAME_SERVER_TYPE;
    void refreshVersionsForType(type, false);
  }, [open, serverType, mcVersionsLoading, refreshVersionsForType]);

  useEffect(() => {
    if (open && !wasOpenRef.current) {
      form.setFieldsValue({
        server_type: DEFAULT_GAME_SERVER_TYPE,
        mc_version: '',
        loader_version: '',
        address: defaultAddress,
        port: suggestDefaultGamePort(existingGames),
        show_in_monitoring: false,
        monitoring_description: '',
        banner_url: '',
        monitoring_tags: [],
      });
    }
    if (!open) {
      versionsLoadSeqRef.current += 1;
    }
    wasOpenRef.current = open;
  }, [open, defaultAddress, existingGames, form]);

  const handleClose = () => {
    form.resetFields();
    onClose();
  };

  const onFinish = async (values: {
    name: string;
    server_type: VpsGameServerType;
    mc_version: string;
    loader_version?: string;
    address: string;
    port: number;
    show_in_monitoring?: boolean;
    monitoring_description?: string;
    banner_url?: string;
    monitoring_tags?: string[];
  }) => {
    setSubmitting(true);
    try {
      await addVpsGameServer(vpsId, {
        name: values.name,
        server_type: values.server_type,
        mc_version: values.mc_version,
        loader_version: gameServerTypeNeedsLoader(values.server_type)
          ? values.loader_version?.trim() || undefined
          : undefined,
        address: values.address,
        port: values.port,
        show_in_monitoring: values.show_in_monitoring,
        monitoring_description: values.monitoring_description?.trim(),
        banner_url: values.banner_url?.trim(),
        monitoring_tags: values.monitoring_tags,
      });
      message.success(t('servers.gameServerCreated'));
      form.resetFields();
      onCreated();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      className="servers-game-modal"
      title={
        <span className="servers-game-modal-title">
          <RocketOutlined aria-hidden />
          {t('servers.addGameServer')}
        </span>
      }
      open={open}
      onCancel={handleClose}
      footer={null}
      width={560}
      destroyOnHidden
      {...modalMotionProps}
    >
      <Paragraph className="servers-game-modal-hint">{t('servers.gameServerModalHint')}</Paragraph>

      <Form form={form} layout="vertical" onFinish={(values) => void onFinish(values)}>
        <div className="servers-game-modal-section">
          <Text className="servers-game-modal-section-label">{t('servers.gameServerSectionBasics')}</Text>
          <Form.Item
            name="name"
            label={t('servers.gameServerName')}
            rules={[{ required: true, message: t('servers.gameServerNameRequired') }]}
          >
            <Input placeholder={t('servers.gameServerNamePlaceholder')} maxLength={64} />
          </Form.Item>
          <Form.Item
            name="server_type"
            label={t('servers.gameServerTypeLabel')}
            rules={[{ required: true, message: t('servers.gameServerTypeRequired') }]}
          >
            <GameServerTypeField />
          </Form.Item>
          <div className="servers-game-modal-type-row">
            <Form.Item
              name="mc_version"
              label={t(gameServerPrimaryVersionI18nKey(serverType))}
              rules={[{ required: true, message: t(gameServerPrimaryVersionRequiredI18nKey(serverType)) }]}
              className="servers-game-modal-type-col"
            >
              <McVersionSelect
                options={mcVersionOptions}
                loading={mcOptionsLoading || mcVersionsLoading}
                placeholder={t(gameServerPrimaryVersionRequiredI18nKey(serverType))}
                onMcVersionChange={handleMcVersionChange}
              />
            </Form.Item>
            {needsLoader ? (
              <Form.Item
                name="loader_version"
                label={gameServerCoreVersionLabel(t, serverType)}
                rules={[{ required: true, message: t('servers.gameServerCoreVersionRequired') }]}
                className="servers-game-modal-type-col"
              >
                <LoaderVersionSelect
                  options={loaderVersionOptions}
                  loading={loaderOptionsLoading}
                  disabled={mcOptionsLoading || loaderVersionOptions.length === 0}
                  placeholder={t('servers.gameServerCoreVersionRequired')}
                />
              </Form.Item>
            ) : (
              <div className="servers-game-modal-type-col" />
            )}
          </div>
          {serverType ? (
            <Paragraph type="secondary" className="servers-game-type-hint">
              {t(`servers.gameServerTypeHint.${serverType}`)}
            </Paragraph>
          ) : null}
        </div>

        <Divider className="servers-game-modal-divider" />

        <div className="servers-game-modal-section">
          <Text className="servers-game-modal-section-label">{t('servers.gameServerSectionNetwork')}</Text>
          <div className="servers-game-modal-network">
            <Form.Item
              name="address"
              label={t('servers.gameServerAddress')}
              rules={[{ required: true, message: t('servers.gameServerAddressRequired') }]}
              className="servers-game-modal-network-address"
            >
              <Input
                prefix={<GlobalOutlined className="servers-game-modal-input-icon" aria-hidden />}
                placeholder={defaultAddress}
              />
            </Form.Item>
            <Form.Item
              name="port"
              label={t('servers.gameServerPort')}
              rules={[{ required: true, message: t('servers.gameServerPortRequired') }]}
              extra={t('servers.gameServerPortHint')}
              className="servers-game-modal-network-port"
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>
          </div>
        </div>

        <Divider className="servers-game-modal-divider" />

        <div className="servers-game-modal-section">
          <Text className="servers-game-modal-section-label">{t('monitoring.monitoringSection')}</Text>
          <Form.Item name="show_in_monitoring" valuePropName="checked">
            <Checkbox>{t('monitoring.showInMonitoring')}</Checkbox>
          </Form.Item>
          <Paragraph type="secondary" className="servers-game-modal-hint">
            {t('monitoring.showInMonitoringHint')}
          </Paragraph>
          {showInMonitoring ? (
            <>
              <Form.Item name="monitoring_description" label={t('monitoring.monitoringDescription')}>
                <Input.TextArea rows={3} maxLength={500} showCount />
              </Form.Item>
              <Form.Item
                name="banner_url"
                label={t('monitoring.monitoringBanner')}
                extra={t('monitoring.monitoringBannerHint')}
              >
                <Input placeholder="https://example.com/banner.png" />
              </Form.Item>
              <Form.Item
                name="monitoring_tags"
                label={t('monitoring.monitoringTags')}
                extra={t('monitoring.monitoringTagsHint')}
              >
                <Select mode="tags" tokenSeparators={[',']} placeholder={t('monitoring.monitoringTags')} />
              </Form.Item>
            </>
          ) : null}
        </div>

        {serverType &&
        (gameServerTypeCapabilities(serverType).plugins || gameServerTypeCapabilities(serverType).mods) ? (
          <Alert
            type="info"
            showIcon
            title={t('servers.gameServerContentNote', {
              content: [
                gameServerTypeCapabilities(serverType).plugins ? t('servers.gameServerContentPlugins') : '',
                gameServerTypeCapabilities(serverType).mods ? t('servers.gameServerContentMods') : '',
              ]
                .filter(Boolean)
                .join(' · '),
            })}
            className="servers-game-modal-note"
          />
        ) : null}

        <div className="servers-game-modal-actions">
          <Button onClick={handleClose}>{t('common.cancel')}</Button>
          <Button type="primary" htmlType="submit" loading={submitting} icon={<PlusOutlined />}>
            {t('servers.addGameServer')}
          </Button>
        </div>
      </Form>
    </Modal>
  );
}
