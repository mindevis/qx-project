import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Button, Form, Input, Modal, Tabs } from 'antd';
import { useAuth } from '@/auth/AuthContext';
import type { AuthMode } from '@/auth/AuthModalContext';

type LoginFormValues = {
  email: string;
  password: string;
};

type RegisterFormValues = {
  email: string;
  password: string;
  confirmPassword: string;
};

type AuthModalProps = {
  open: boolean;
  mode: AuthMode;
  onModeChange: (mode: AuthMode) => void;
  onClose: () => void;
};

export function AuthModal({ open, mode, onModeChange, onClose }: AuthModalProps) {
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const [loginForm] = Form.useForm<LoginFormValues>();
  const [registerForm] = Form.useForm<RegisterFormValues>();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleClose = () => {
    setError(null);
    loginForm.resetFields();
    registerForm.resetFields();
    onClose();
  };

  const handleSuccess = () => {
    handleClose();
    navigate('/profile');
  };

  const onLoginFinish = async (values: LoginFormValues) => {
    setSubmitting(true);
    setError(null);
    try {
      await login(values.email, values.password);
      handleSuccess();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка входа');
    } finally {
      setSubmitting(false);
    }
  };

  const onRegisterFinish = async (values: RegisterFormValues) => {
    setSubmitting(true);
    setError(null);
    try {
      await register(values.email, values.password);
      handleSuccess();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка регистрации');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="Аккаунт"
      open={open}
      onCancel={handleClose}
      footer={null}
      destroyOnClose
      width={420}
    >
      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}
      <Tabs
        activeKey={mode}
        destroyInactiveTabPane
        onChange={(key) => {
          setError(null);
          onModeChange(key as AuthMode);
        }}
        items={[
          {
            key: 'login',
            label: 'Вход',
            children: (
              <Form form={loginForm} layout="vertical" onFinish={onLoginFinish}>
                <Form.Item
                  label="Email"
                  name="email"
                  rules={[{ required: true, type: 'email', message: 'Введите email' }]}
                >
                  <Input autoComplete="email" />
                </Form.Item>
                <Form.Item
                  label="Пароль"
                  name="password"
                  rules={[{ required: true, message: 'Введите пароль' }]}
                >
                  <Input.Password autoComplete="current-password" />
                </Form.Item>
                <Button type="primary" htmlType="submit" block loading={submitting}>
                  Войти
                </Button>
              </Form>
            ),
          },
          {
            key: 'register',
            label: 'Регистрация',
            children: (
              <Form form={registerForm} layout="vertical" onFinish={onRegisterFinish}>
                <Form.Item
                  label="Email"
                  name="email"
                  rules={[{ required: true, type: 'email', message: 'Введите email' }]}
                >
                  <Input autoComplete="email" />
                </Form.Item>
                <Form.Item
                  label="Пароль"
                  name="password"
                  rules={[
                    { required: true, message: 'Введите пароль' },
                    { min: 8, message: 'Минимум 8 символов' },
                  ]}
                >
                  <Input.Password autoComplete="new-password" />
                </Form.Item>
                <Form.Item
                  label="Повтор пароля"
                  name="confirmPassword"
                  dependencies={['password']}
                  rules={[
                    { required: true, message: 'Повторите пароль' },
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!value || getFieldValue('password') === value) {
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
                  Создать аккаунт
                </Button>
              </Form>
            ),
          },
        ]}
      />
    </Modal>
  );
}
