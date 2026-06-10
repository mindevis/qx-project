import { useState } from 'react';
import { Alert, Button, Form, Input, Modal } from 'antd';
import { api } from '@/api/client';

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
      setError(e instanceof Error ? e.message : 'Не удалось сменить пароль');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="Смена пароля"
      open={open}
      onCancel={handleClose}
      footer={null}
      destroyOnClose
      width={420}
    >
      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          label="Текущий пароль"
          name="current_password"
          rules={[{ required: true, message: 'Введите текущий пароль' }]}
        >
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Form.Item
          label="Новый пароль"
          name="new_password"
          rules={[
            { required: true, message: 'Введите новый пароль' },
            { min: 8, message: 'Минимум 8 символов' },
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          label="Повтор нового пароля"
          name="confirm_password"
          dependencies={['new_password']}
          rules={[
            { required: true, message: 'Повторите новый пароль' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('new_password') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('Пароли не совпадают'));
              },
            }),
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={submitting}>
          Сохранить
        </Button>
      </Form>
    </Modal>
  );
}
