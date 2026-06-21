import { useState } from 'react';
import { Alert, Button, Form, Input, Modal, Typography } from 'antd';
import { api } from '@/api/client';

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
  const [form] = Form.useForm<EmailFormValues>();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

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
      setError(e instanceof Error ? e.message : 'Не удалось сменить email');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="Смена email"
      open={open}
      onCancel={handleClose}
      footer={null}
      destroyOnHidden
      width={420}
      afterOpenChange={(visible) => {
        if (visible) {
          form.setFieldsValue({ email: currentEmail });
        }
      }}
    >
      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}
      <Typography.Paragraph type="secondary">
        Для подтверждения потребуется текущий пароль.
      </Typography.Paragraph>
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          label="Новый email"
          name="email"
          rules={[{ required: true, type: 'email', message: 'Введите корректный email' }]}
        >
          <Input autoComplete="email" />
        </Form.Item>
        <Form.Item
          label="Текущий пароль"
          name="current_password"
          rules={[{ required: true, message: 'Введите текущий пароль' }]}
        >
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={submitting}>
          Сохранить
        </Button>
      </Form>
    </Modal>
  );
}
