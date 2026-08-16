import { Alert, Button, Modal, Progress, Steps, Typography } from 'antd';
import { useI18n } from '@/i18n/I18nContext';
import { getLaunchStatusKey } from '@/i18n';
import { getLaunchErrorKey } from '@/lib/launchProgress';
import { modalMotionProps } from '@/lib/modal';
import {
  CONNECT_STEPS,
  connectFileProgressKey,
  type ConnectProgressStep,
} from '@/lib/connectProgress';
import type { ReactNode } from 'react';
import './ConnectProgressModal.css';

const { Paragraph, Text } = Typography;

export type ConnectProgressModalProps = {
  open: boolean;
  serverName: string;
  step: ConnectProgressStep;
  status?: string;
  detail?: string;
  errorCode?: string;
  failed?: boolean;
  onClose: () => void;
  children?: ReactNode;
};

export function ConnectProgressModal({
  open,
  serverName,
  step,
  status,
  detail,
  errorCode,
  failed,
  onClose,
  children,
}: ConnectProgressModalProps) {
  const { t } = useI18n();
  const current = CONNECT_STEPS.indexOf(step);
  const fileKey = connectFileProgressKey(detail);
  const fileLabel = fileKey ? t(fileKey) : detail;
  const statusKey = status ? getLaunchStatusKey(status) : '';
  const statusLabel = statusKey ? t(statusKey) : status;
  const errorKey = errorCode ? getLaunchErrorKey(errorCode) : undefined;
  const errorLabel = errorKey ? t(errorKey) : errorCode;

  return (
    <Modal
      {...modalMotionProps}
      title={t('monitoring.connectProgress.title', { server: serverName })}
      open={open}
      onCancel={onClose}
      footer={
        failed
          ? [
              <Button key="close" type="primary" onClick={onClose}>
                {t('common.close')}
              </Button>,
            ]
          : step === 'clientMods'
            ? null
            : [
                <Button key="cancel" onClick={onClose}>
                  {t('common.cancel')}
                </Button>,
              ]
      }
      maskClosable={failed}
      closable
      width={560}
    >
      <Steps
        size="small"
        current={current}
        status={failed ? 'error' : 'process'}
        className="connect-progress-steps"
        items={CONNECT_STEPS.map((id) => ({
          title: t(`monitoring.connectProgress.steps.${id}`),
        }))}
      />

      {failed ? (
        <Alert
          type="error"
          showIcon
          className="connect-progress-alert"
          message={t('monitoring.connectProgress.failedTitle')}
          description={
            <div>
              {errorLabel ? <Paragraph>{errorLabel}</Paragraph> : null}
              {fileLabel ? <Text type="secondary">{fileLabel}</Text> : null}
            </div>
          }
        />
      ) : (
        <div className="connect-progress-body">
          <Progress percent={Math.round(((current + 1) / CONNECT_STEPS.length) * 100)} status="active" />
          <Paragraph className="connect-progress-status">
            {t(`monitoring.connectProgress.detail.${step}`)}
          </Paragraph>
          {statusLabel ? (
            <Text type="secondary" className="connect-progress-sub">
              {statusLabel}
            </Text>
          ) : null}
          {fileLabel ? (
            <Text className="connect-progress-file">
              {t('monitoring.connectProgress.nowDoing', { what: fileLabel })}
            </Text>
          ) : null}
          {step === 'clientMods' ? children : null}
        </div>
      )}
    </Modal>
  );
}
