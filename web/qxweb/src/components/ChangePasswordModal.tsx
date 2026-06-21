import { useState } from 'react';
import { Alert, Button, Form, Input, Modal } from 'antd';
import { api } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { modalMotionProps } from '@/lib/modal';

type PasswordFormValues = {
  current_password: string;
  new_password: string;
  confirm_password: string;
};

type ChangePasswordModalProps = {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
};

export function ChangePasswordModal({ open, onClose, onSuccess }: ChangePasswordModalProps) {
  const { t } = useI18n();
  const [form] = Form.useForm<PasswordFormValues>();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleClose = () => {
    setError(null);
    form.resetFields();
    onClose();
  };

  const onFinish = async (values: PasswordFormValues) => {
    setSubmitting(true);
    setError(null);
    try {
      await api.changePassword({
        current_password: values.current_password,
        new_password: values.new_password,
      });
      handleClose();
      onSuccess();
    } catch (e) {
      setError(e instanceof Error ? e.message : t('profile.changePasswordError'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('profile.changePasswordTitle')}
      open={open}
      onCancel={handleClose}
      footer={null}
      destroyOnHidden
      width={420}
      {...modalMotionProps}
    >
      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          label={t('profile.currentPassword')}
          name="current_password"
          rules={[{ required: true, message: t('auth.passwordRequired') }]}
        >
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Form.Item
          label={t('profile.newPassword')}
          name="new_password"
          rules={[
            { required: true, message: t('profile.newPasswordRequired') },
            { min: 8, message: t('auth.passwordMin8') },
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          label={t('profile.confirmNewPassword')}
          name="confirm_password"
          dependencies={['new_password']}
          rules={[
            { required: true, message: t('profile.confirmNewPasswordRequired') },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('new_password') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error(t('auth.passwordsMismatch')));
              },
            }),
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={submitting}>
          {t('common.save')}
        </Button>
      </Form>
    </Modal>
  );
}
