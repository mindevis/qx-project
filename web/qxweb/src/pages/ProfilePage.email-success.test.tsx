import { useEffect, useRef } from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { renderWithProviders } from '@/test/test-utils';
import { saveTokens } from '@/api/client';

vi.mock('skinview3d', async () => {
  const { skinview3dWebGlFailureMock } = await import('@/test/skinview3d-mock');
  return skinview3dWebGlFailureMock;
});

vi.mock('@/components/ChangeEmailModal', () => ({
  ChangeEmailModal: ({ onSuccess, open }: { onSuccess: () => void; open: boolean }) => {
    const firedRef = useRef(false);
    const onSuccessRef = useRef(onSuccess);
    onSuccessRef.current = onSuccess;
    useEffect(() => {
      if (!open) {
        firedRef.current = false;
        return;
      }
      if (!firedRef.current) {
        firedRef.current = true;
        queueMicrotask(() => onSuccessRef.current());
      }
    }, [open]);
    return null;
  },
}));

import { ProfilePage } from './ProfilePage';

function mockProfileFetch() {
  vi.mocked(fetch).mockImplementation((input: RequestInfo | URL) => {
    const url =
      typeof input === 'string'
        ? input
        : input instanceof URL
          ? input.href
          : input.url;
    if (url.includes('/auth/refresh')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            access_token: 'a',
            refresh_token: 'r',
            token_type: 'Bearer',
            expires_in: 3600,
          }),
          { status: 200 },
        ),
      );
    }
    if (url.includes('/users/me/mojang')) {
      return Promise.resolve(
        new Response(JSON.stringify({ linked: false }), { status: 200 }),
      );
    }
    if (url.includes('/users/me/cosmetics')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            skin_model: 'steve',
            has_skin: false,
            has_cape: false,
            updated_at: '2026-01-01T00:00:00Z',
          }),
          { status: 200 },
        ),
      );
    }
    if (url.includes('/users/me')) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'user@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      );
    }
    return Promise.resolve(new Response('{}', { status: 200 }));
  });
}

describe('ProfilePage email success', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
      saved_at: Date.now(),
    });
    mockProfileFetch();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('shows email success message after modal completes', async () => {
    const user = userEvent.setup({ delay: null });
  const successSpy = testMessage.success;

    renderWithProviders(<ProfilePage />, '/profile');
    await waitFor(() => expect(screen.getByText('user@test.com')).toBeInTheDocument());

    await user.click(screen.getByLabelText('Сменить email'));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Email изменён'));
  });
});
