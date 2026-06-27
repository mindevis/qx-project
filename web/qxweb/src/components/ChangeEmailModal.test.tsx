import type { ComponentProps } from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { saveTokens, api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { ChangeEmailModal } from './ChangeEmailModal';

const tokens = {
  access_token: 'access',
  refresh_token: 'refresh',
  token_type: 'Bearer',
  expires_in: 60,
};

function renderModal(props: Partial<ComponentProps<typeof ChangeEmailModal>> = {}) {
  return renderWithTheme(
    <ChangeEmailModal
        open
        currentEmail="old@test.com"
        onClose={vi.fn()}
        onSuccess={vi.fn()}
        {...props}
    />,
  );
}

describe('ChangeEmailModal', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens(tokens);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('submits email change and closes on success', async () => {
    const user = userEvent.setup({ delay: null });
    const onClose = vi.fn();
    const onSuccess = vi.fn();
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: '1',
          email: 'new@test.com',
          tier: 'free',
          created_at: 'now',
        }),
        { status: 200 },
      ),
    );

    renderModal({ onClose, onSuccess });

    await user.clear(screen.getByLabelText('Новый email'));
    await user.type(screen.getByLabelText('Новый email'), 'new@test.com');
    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('shows error when api fails', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: 'X', message: 'email taken' } }), {
        status: 422,
      }),
    );

    renderModal();
    await waitFor(() => expect(screen.getByLabelText('Новый email')).toHaveValue('old@test.com'));

    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(screen.getByText('email taken')).toBeInTheDocument());
  });

  it('shows generic error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'changeEmail').mockRejectedValueOnce('fail');

    renderModal();
    await waitFor(() => expect(screen.getByLabelText('Новый email')).toHaveValue('old@test.com'));

    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(screen.getByText('Не удалось сменить email')).toBeInTheDocument());
  });
});
