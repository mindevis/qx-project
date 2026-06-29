import type { ComponentProps } from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { saveTokens, api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { ChangePasswordModal } from './ChangePasswordModal';

const tokens = {
  access_token: 'access',
  refresh_token: 'refresh',
  token_type: 'Bearer',
  expires_in: 3600,
};

function renderModal(props: Partial<ComponentProps<typeof ChangePasswordModal>> = {}) {
  return renderWithTheme(
    <ChangePasswordModal open onClose={vi.fn()} onSuccess={vi.fn()} {...props} />,
  );
}

describe('ChangePasswordModal', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens(tokens);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('submits password change and closes on success', async () => {
    const user = userEvent.setup({ delay: null });
    const onClose = vi.fn();
    const onSuccess = vi.fn();
    vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 204 }));

    renderModal({ onClose, onSuccess });

    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.type(screen.getByLabelText('Новый пароль'), 'newpassword456');
    await user.type(screen.getByLabelText('Повтор нового пароля'), 'newpassword456');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('shows error when api fails', async () => {
    const user = userEvent.setup({ delay: null });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: 'sX', message: 'bad password' } }), {
        status: 400,
      }),
    );

    renderModal();

    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.type(screen.getByLabelText('Новый пароль'), 'newpassword456');
    await user.type(screen.getByLabelText('Повтор нового пароля'), 'newpassword456');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(screen.getByText('bad password')).toBeInTheDocument());
  });

  it('shows generic error for non-error throws', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'changePassword').mockRejectedValueOnce('fail');

    renderModal();

    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.type(screen.getByLabelText('Новый пароль'), 'newpassword456');
    await user.type(screen.getByLabelText('Повтор нового пароля'), 'newpassword456');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(screen.getByText('Не удалось сменить пароль')).toBeInTheDocument());
  });

  it('resets form on cancel', async () => {
    const user = userEvent.setup({ delay: null });
    const onClose = vi.fn();

    renderModal({ onClose });

    await user.type(screen.getByLabelText('Текущий пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalled();
  });
});
