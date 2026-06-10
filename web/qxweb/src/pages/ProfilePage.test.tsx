import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { renderWithProviders } from '@/test/test-utils';
import { ProfilePage } from './ProfilePage';
import { saveTokens } from '@/api/client';

describe('ProfilePage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
    vi.mocked(fetch).mockResolvedValue(
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
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('opens email and password modals', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<ProfilePage />, '/profile');

    await waitFor(() => expect(screen.getByText('user@test.com')).toBeInTheDocument());

    await user.click(screen.getByLabelText('Сменить email'));
    await waitFor(() => expect(screen.getByText('Смена email')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Close' }));

    await user.click(screen.getByRole('button', { name: 'Сменить пароль' }));
    await waitFor(() => expect(screen.getByText('Смена пароля')).toBeInTheDocument());
  });

  it('shows success message after email change', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');
    const profile = (email: string) =>
      new Response(
        JSON.stringify({
          id: '1',
          email,
          tier: 'free',
          created_at: 'now',
        }),
        { status: 200 },
      );
    vi.mocked(fetch).mockImplementation((_url, init) => {
      const method = init?.method ?? 'GET';
      if (method === 'PATCH') {
        return Promise.resolve(profile('new@test.com'));
      }
      return Promise.resolve(profile('user@test.com'));
    });

    renderWithProviders(<ProfilePage />, '/profile');
    await waitFor(() => expect(screen.getByText('user@test.com')).toBeInTheDocument());

    await user.click(screen.getByLabelText('Сменить email'));
    const dialog = await screen.findByRole('dialog');
    await user.clear(within(dialog).getByLabelText('Новый email'));
    await user.type(within(dialog).getByLabelText('Новый email'), 'new@test.com');
    await user.type(within(dialog).getByLabelText('Текущий пароль'), 'password123');
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Email изменён'));
    successSpy.mockRestore();
  });

  it('shows success message after password change', async () => {
    const user = userEvent.setup({ delay: null });
    const successSpy = vi.spyOn(message, 'success');
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: '1',
            email: 'user@test.com',
            tier: 'free',
            created_at: 'now',
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));

    renderWithProviders(<ProfilePage />, '/profile');
    await waitFor(() => expect(screen.getByText('user@test.com')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: 'Сменить пароль' }));
    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText('Текущий пароль'), 'password123');
    await user.type(within(dialog).getByLabelText('Новый пароль'), 'newpassword456');
    await user.type(within(dialog).getByLabelText('Повтор нового пароля'), 'newpassword456');
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(successSpy).toHaveBeenCalledWith('Пароль изменён'));
    successSpy.mockRestore();
  });
});
