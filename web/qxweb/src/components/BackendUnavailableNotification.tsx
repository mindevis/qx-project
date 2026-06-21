import { useEffect } from 'react';
import { App } from 'antd';
import { useBackendStatus } from '@/backend/BackendStatusContext';
import { useI18n } from '@/i18n/I18nContext';

export const BACKEND_UNAVAILABLE_NOTIFICATION_KEY = 'backend-unavailable';

export function BackendUnavailableNotification() {
  const { available } = useBackendStatus();
  const { notification } = App.useApp();
  const { t } = useI18n();

  useEffect(() => {
    if (available) {
      notification.destroy(BACKEND_UNAVAILABLE_NOTIFICATION_KEY);
      return;
    }

    notification.error({
      key: BACKEND_UNAVAILABLE_NOTIFICATION_KEY,
      message: t('backend.title'),
      description: t('backend.description'),
      duration: 0,
      placement: 'bottomRight',
    });
  }, [available, notification, t]);

  useEffect(
    () => () => notification.destroy(BACKEND_UNAVAILABLE_NOTIFICATION_KEY),
    [notification],
  );

  return null;
}
