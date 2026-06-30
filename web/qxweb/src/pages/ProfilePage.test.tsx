import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { renderWithProviders } from '@/test/test-utils';
import { ProfilePage } from './ProfilePage';
import { saveTokens, api } from '@/api/client';

vi.mock('skinview3d', async () => {
  const { skinview3dMock } = await import('@/test/skinview3d-mock');
  return skinview3dMock;
});

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

describe('ProfilePage', () => {
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

  it('shows mojang link section when not linked', async () => {
    renderWithProviders(<ProfilePage />, '/profile');
    await waitFor(() =>
      expect(screen.getByText('Аккаунт Minecraft (Microsoft)')).toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Привязать Microsoft/i })).toBeInTheDocument(),
    );
  });

  it('starts mojang oauth when link button clicked', async () => {
    const assign = vi.fn();
    vi.stubGlobal('location', { ...window.location, assign });
    vi.spyOn(api, 'startMojangOAuth').mockResolvedValue({
      authorization_url: 'https://login.microsoftonline.com/oauth',
    });
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<ProfilePage />, '/profile');
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Привязать Microsoft/i })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /Привязать Microsoft/i }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith('https://login.microsoftonline.com/oauth'));
    vi.unstubAllGlobals();
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
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ linked: false }), { status: 200 }),
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
