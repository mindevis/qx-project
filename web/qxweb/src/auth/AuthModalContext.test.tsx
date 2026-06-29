import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { AuthModalProvider, useAuthModal } from '@/auth/AuthModalContext';
import { waitForNoDialog, renderWithTheme } from '@/test/test-utils';

function Probe() {
  const { openAuthModal, closeAuthModal } = useAuthModal();
  return (
    <>
      <button type="button" onClick={() => openAuthModal('login')}>
        Open login
      </button>
      <button type="button" onClick={() => openAuthModal('register')}>
        Open register
      </button>
      <button type="button" onClick={closeAuthModal}>
        Close modal
      </button>
    </>
  );
}

function renderProbe() {
  return renderWithTheme(
    <MemoryRouter>
      <AuthProvider>
        <AuthModalProvider>
          <Probe />
        </AuthModalProvider>
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe('AuthModalContext', () => {
  it('opens and closes auth modal', async () => {
    const user = userEvent.setup({ delay: null });
    renderProbe();

    await user.click(screen.getByRole('button', { name: 'Open login' }));
    await screen.findByRole('dialog');

    await user.click(screen.getByRole('button', { name: 'Close modal' }));
    await waitForNoDialog();
  });

  it('opens register tab', async () => {
    const user = userEvent.setup({ delay: null });
    renderProbe();

    await user.click(screen.getByRole('button', { name: 'Open register' }));
    await waitFor(() => expect(screen.getByRole('tab', { name: 'Регистрация' })).toHaveAttribute('aria-selected', 'true'));
  });

  it('returns to the page where sign-in was opened', async () => {
    const user = userEvent.setup({ delay: null });
    vi.stubGlobal('fetch', vi.fn());
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            access_token: 'a',
            refresh_token: 'r',
            token_type: 'Bearer',
            expires_in: 3600,
          }),
          { status: 200 },
        ),
      )
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
      );

    function LauncherProbe() {
      const { openAuthModal } = useAuthModal();
      return (
        <>
          <p>Launcher workspace</p>
          <button type="button" onClick={() => openAuthModal('login')}>
            Sign in here
          </button>
        </>
      );
    }

    renderWithTheme(
      <MemoryRouter initialEntries={['/launcher']}>
        <AuthProvider>
          <AuthModalProvider>
            <Routes>
              <Route path="/launcher" element={<LauncherProbe />} />
            </Routes>
          </AuthModalProvider>
        </AuthProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole('button', { name: 'Sign in here' }));
    await screen.findByRole('dialog');
    await user.type(screen.getByLabelText('Email'), 'user@test.com');
    await user.type(screen.getByLabelText('Пароль'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Войти' }));

    await waitForNoDialog();
    expect(screen.getByText('Launcher workspace')).toBeInTheDocument();
    vi.unstubAllGlobals();
  });

  it('throws outside provider', () => {
    expect(() => render(<Probe />)).toThrow('useAuthModal must be used within AuthModalProvider');
  });
});
