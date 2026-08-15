import { useEffect, useState } from 'react';
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
import { EditOutlined, GlobalOutlined } from '@ant-design/icons';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { updateVpsGameServer, type VpsGameServer } from '@/lib/vpsGameServers';
import { logger } from '@/lib/logger';
import '@/pages/ServersPage.css';

const { Text } = Typography;

export function EditGameServerModal({
  open,
  vpsId,
  game,
  existingGames,
  onClose,
  onUpdated,
}: {
  open: boolean;
  vpsId: string;
  game: VpsGameServer | null;
  existingGames: VpsGameServer[];
  onClose: () => void;
  onUpdated: (updated: VpsGameServer) => void;
}) {
  const { t } = useI18n();
  const message = useMessage();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const showInMonitoring = Form.useWatch('show_in_monitoring', form) as boolean | undefined;

  useEffect(() => {
    if (open && game) {
      form.setFieldsValue({
        name: game.name,
        address: game.address ?? '',
        port: game.port,
        show_in_monitoring: game.show_in_monitoring ?? false,
        monitoring_description: game.monitoring_description ?? '',
        banner_url: game.banner_url ?? '',
        monitoring_tags: game.monitoring_tags ?? [],
      });
    }
  }, [open, game, form]);

  const handleClose = () => {
    form.resetFields();
    onClose();
  };

  const onFinish = async (values: {
    name: string;
    address?: string;
    port: number;
    show_in_monitoring?: boolean;
    monitoring_description?: string;
    banner_url?: string;
    monitoring_tags?: string[];
  }) => {
    if (!game) return;
    const port = values.port;
    const portTaken = existingGames.some(
      (item) => item.id !== game.id && item.port === port,
    );
    if (portTaken) {
      message.warning(t('servers.gameServerPortInUse'));
      return;
    }
    setSubmitting(true);
    try {
      const updated = await updateVpsGameServer(vpsId, game.id, {
        name: values.name.trim(),
        address: values.address?.trim() ?? '',
        port,
        show_in_monitoring: values.show_in_monitoring,
        monitoring_description: values.monitoring_description?.trim(),
        banner_url: values.banner_url?.trim(),
        monitoring_tags: values.monitoring_tags,
      });
      message.success(t('servers.gameServerUpdated'));
      onUpdated(updated);
      handleClose();
    } catch (e) {
      logger.warn('failed to update game server', { error: String(e) });
      message.error(t('servers.gameServerUpdateFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      open={open}
      title={t('servers.editGameServer')}
      onCancel={handleClose}
      footer={null}
      destroyOnHidden
      className="servers-game-modal"
    >
      <Form form={form} layout="vertical" onFinish={(values) => void onFinish(values)}>
        <Form.Item
          name="name"
          label={t('servers.gameServerName')}
          rules={[{ required: true, message: t('servers.gameServerNameRequired') }]}
        >
          <Input />
        </Form.Item>
        <Form.Item name="address" label={t('servers.gameServerAddress')}>
          <Input prefix={<GlobalOutlined className="servers-game-modal-input-icon" aria-hidden />} />
        </Form.Item>
        <Form.Item
          name="port"
          label={t('servers.gameServerPort')}
          rules={[{ required: true, message: t('servers.gameServerPortRequired') }]}
          extra={t('servers.gameServerEditPortHint')}
        >
          <InputNumber min={1} max={65535} className="servers-game-port-input" />
        </Form.Item>

        <Divider className="servers-game-modal-divider" />

        <Text className="servers-game-modal-section-label">{t('monitoring.monitoringSection')}</Text>
        <Form.Item name="show_in_monitoring" valuePropName="checked">
          <Checkbox>{t('monitoring.showInMonitoring')}</Checkbox>
        </Form.Item>
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

        <Alert
          type="info"
          showIcon
          title={t('servers.gameServerEditHint')}
          className="servers-game-modal-note"
        />
        <div className="servers-game-modal-actions">
          <Button onClick={handleClose}>{t('common.cancel')}</Button>
          <Button type="primary" htmlType="submit" loading={submitting} icon={<EditOutlined />}>
            {t('common.save')}
          </Button>
        </div>
      </Form>
    </Modal>
  );
}
