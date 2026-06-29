import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { renderWithProviders } from '@/test/test-utils';
import { saveTokens } from '@/api/client';

vi.mock('@/components/ChangeEmailModal', () => ({
  ChangeEmailModal: ({ onSuccess, open }: { onSuccess: () => void; open: boolean }) => {
    if (!open) return null;
    queueMicrotask(() => onSuccess());
    return null;
  },
}));

import { ProfilePage } from './ProfilePage';

describe('ProfilePage email success', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 3600,
    });
    vi.mocked(fetch).mockImplementation(() =>
      Promise.resolve(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'user@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      ),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('shows email success message after modal completes', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');

    renderWithProviders(<ProfilePage />, '/profile');
    await waitFor(() => expect(screen.getByText('user@test.com')).toBeInTheDocument());

    await user.click(screen.getByLabelText('Сменить email'));
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Email изменён'));
    successSpy.mockRestore();
  });
});
