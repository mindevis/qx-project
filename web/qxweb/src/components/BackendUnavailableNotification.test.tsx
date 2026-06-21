import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { App } from 'antd';
import { I18nProvider } from '@/i18n/I18nContext';
import {
  BACKEND_UNAVAILABLE_NOTIFICATION_KEY,
  BackendUnavailableNotification,
} from './BackendUnavailableNotification';

vi.mock('@/backend/BackendStatusContext', () => ({
  useBackendStatus: vi.fn(),
}));

import { useBackendStatus } from '@/backend/BackendStatusContext';

describe('BackendUnavailableNotification', () => {
  const destroy = vi.fn();
  const error = vi.fn();

  beforeEach(() => {
    destroy.mockReset();
    error.mockReset();
    vi.spyOn(App, 'useApp').mockReturnValue({
      notification: { destroy, error },
    } as ReturnType<typeof App.useApp>);
  });

  function renderNotification() {
    return render(
      <I18nProvider>
        <BackendUnavailableNotification />
      </I18nProvider>,
    );
  }

  it('destroys notification when backend is available', () => {
    vi.mocked(useBackendStatus).mockReturnValue({ available: true });

    renderNotification();

    expect(error).not.toHaveBeenCalled();
    expect(destroy).toHaveBeenCalledWith(BACKEND_UNAVAILABLE_NOTIFICATION_KEY);
  });

  it('opens persistent antd notification when backend is unavailable', () => {
    vi.mocked(useBackendStatus).mockReturnValue({ available: false });

    renderNotification();

    expect(error).toHaveBeenCalledWith({
      key: BACKEND_UNAVAILABLE_NOTIFICATION_KEY,
      message: 'Сервер недоступен',
      description:
        'Не удаётся связаться с API. Некоторые функции недоступны — ожидаем восстановление…',
      duration: 0,
      placement: 'bottomRight',
    });
  });

  it('destroys notification when backend recovers', () => {
    vi.mocked(useBackendStatus).mockReturnValue({ available: false });

    const { rerender } = renderNotification();

    expect(error).toHaveBeenCalledTimes(1);

    vi.mocked(useBackendStatus).mockReturnValue({ available: true });
    rerender(
      <I18nProvider>
        <BackendUnavailableNotification />
      </I18nProvider>,
    );

    expect(destroy).toHaveBeenCalledWith(BACKEND_UNAVAILABLE_NOTIFICATION_KEY);
  });

  it('destroys notification on unmount', () => {
    vi.mocked(useBackendStatus).mockReturnValue({ available: false });

    const { unmount } = renderNotification();
    destroy.mockClear();

    unmount();

    expect(destroy).toHaveBeenCalledWith(BACKEND_UNAVAILABLE_NOTIFICATION_KEY);
  });
});
