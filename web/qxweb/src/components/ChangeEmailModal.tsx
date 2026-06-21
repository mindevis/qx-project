import { useEffect, useState } from 'react';
import { Alert, Button, Form, Input, Modal, Typography } from 'antd';
import { api } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { modalMotionProps } from '@/lib/modal';

type EmailFormValues = {
  current_password: string;
  email: string;
};

type ChangeEmailModalProps = {
  open: boolean;
  currentEmail: string;
  onClose: () => void;
  onSuccess: () => void;
};

export function ChangeEmailModal({
  open,
  currentEmail,
  onClose,
  onSuccess,
}: ChangeEmailModalProps) {
  const { t } = useI18n();
  const [form] = Form.useForm<EmailFormValues>();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ email: currentEmail });
    }
  }, [open, currentEmail, form]);

  const handleClose = () => {
    setError(null);
    form.resetFields();
    onClose();
  };

  const onFinish = async (values: EmailFormValues) => {
    setSubmitting(true);
    setError(null);
    try {
      await api.changeEmail({
        current_password: values.current_password,
        email: values.email,
      });
      handleClose();
      onSuccess();
    } catch (e) {
      setError(e instanceof Error ? e.message : t('profile.changeEmailError'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('profile.changeEmailTitle')}
      open={open}
      onCancel={handleClose}
      footer={null}
      destroyOnHidden
      width={420}
      {...modalMotionProps}
    >
      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}
      <Typography.Paragraph type="secondary">{t('profile.changeEmailHint')}</Typography.Paragraph>
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          label={t('profile.newEmail')}
          name="email"
          rules={[{ required: true, type: 'email', message: t('profile.emailInvalid') }]}
        >
          <Input autoComplete="email" />
        </Form.Item>
        <Form.Item
          label={t('profile.currentPassword')}
          name="current_password"
          rules={[{ required: true, message: t('auth.passwordRequired') }]}
        >
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={submitting}>
          {t('common.save')}
        </Button>
      </Form>
    </Modal>
  );
}
