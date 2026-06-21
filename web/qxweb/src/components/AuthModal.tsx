import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Button, Form, Input, Modal, Tabs } from 'antd';
import { useAuth } from '@/auth/AuthContext';
import type { AuthMode } from '@/auth/AuthModalContext';
import { useI18n } from '@/i18n/I18nContext';
import { modalMotionProps } from '@/lib/modal';

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
  const { t } = useI18n();
  const navigate = useNavigate();
  const [loginForm] = Form.useForm<LoginFormValues>();
  const [registerForm] = Form.useForm<RegisterFormValues>();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleClose = () => {
    setError(null);
    if (mode === 'login') {
      loginForm.resetFields();
    } else {
      registerForm.resetFields();
    }
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
      setError(e instanceof Error ? e.message : t('auth.loginError'));
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
      setError(e instanceof Error ? e.message : t('auth.registerError'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('auth.account')}
      open={open}
      onCancel={handleClose}
      footer={null}
      destroyOnHidden
      width={420}
      {...modalMotionProps}
    >
      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}
      <Tabs
        activeKey={mode}
        destroyOnHidden
        onChange={(key) => {
          setError(null);
          onModeChange(key as AuthMode);
        }}
        items={[
          {
            key: 'login',
            label: t('auth.loginTab'),
            children: (
              <Form form={loginForm} layout="vertical" onFinish={onLoginFinish}>
                <Form.Item
                  label={t('common.email')}
                  name="email"
                  rules={[{ required: true, type: 'email', message: t('auth.emailRequired') }]}
                >
                  <Input autoComplete="email" />
                </Form.Item>
                <Form.Item
                  label={t('common.password')}
                  name="password"
                  rules={[{ required: true, message: t('auth.passwordRequired') }]}
                >
                  <Input.Password autoComplete="current-password" />
                </Form.Item>
                <Button type="primary" htmlType="submit" block loading={submitting}>
                  {t('auth.signIn')}
                </Button>
              </Form>
            ),
          },
          {
            key: 'register',
            label: t('auth.registerTab'),
            children: (
              <Form form={registerForm} layout="vertical" onFinish={onRegisterFinish}>
                <Form.Item
                  label={t('common.email')}
                  name="email"
                  rules={[{ required: true, type: 'email', message: t('auth.emailRequired') }]}
                >
                  <Input autoComplete="email" />
                </Form.Item>
                <Form.Item
                  label={t('common.password')}
                  name="password"
                  rules={[
                    { required: true, message: t('auth.passwordRequired') },
                    { min: 8, message: t('auth.passwordMin8') },
                  ]}
                >
                  <Input.Password autoComplete="new-password" />
                </Form.Item>
                <Form.Item
                  label={t('auth.confirmPassword')}
                  name="confirmPassword"
                  dependencies={['password']}
                  rules={[
                    { required: true, message: t('auth.confirmPasswordRequired') },
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!value || getFieldValue('password') === value) {
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
                  {t('auth.createAccount')}
                </Button>
              </Form>
            ),
          },
        ]}
      />
    </Modal>
  );
}
