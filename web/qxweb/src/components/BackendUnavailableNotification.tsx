import { useEffect, useRef } from 'react';
import { App } from 'antd';
import { useBackendStatus } from '@/backend/BackendStatusContext';
import { useI18n } from '@/i18n/I18nContext';

export const BACKEND_UNAVAILABLE_NOTIFICATION_KEY = 'backend-unavailable';

export function BackendUnavailableNotification() {
  const { available } = useBackendStatus();
  const { notification } = App.useApp();
  const { t } = useI18n();
  const prevAvailableRef = useRef<boolean | null>(null);

  useEffect(() => {
    const prev = prevAvailableRef.current;
    prevAvailableRef.current = available;

    if (available) {
      if (prev === false) {
        notification.destroy(BACKEND_UNAVAILABLE_NOTIFICATION_KEY);
      }
      return;
    }

    if (prev === null || prev === true) {
      notification.error({
        key: BACKEND_UNAVAILABLE_NOTIFICATION_KEY,
        title: t('backend.title'),
        description: t('backend.description'),
        duration: 0,
        placement: 'bottomRight',
      });
    }
  }, [available, notification, t]);

  useEffect(
    () => () => notification.destroy(BACKEND_UNAVAILABLE_NOTIFICATION_KEY),
    [notification],
  );

  return null;
}
