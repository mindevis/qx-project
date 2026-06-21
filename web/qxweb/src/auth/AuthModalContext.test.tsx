import { describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { AuthModalProvider, useAuthModal } from '@/auth/AuthModalContext';
import { ThemeProvider } from '@/theme/ThemeContext';
import { waitForNoDialog } from '@/test/test-utils';

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
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <AuthProvider>
          <AuthModalProvider>
            <Probe />
          </AuthModalProvider>
        </AuthProvider>
      </MemoryRouter>
    </ThemeProvider>,
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

  it('throws outside provider', () => {
    expect(() => render(<Probe />)).toThrow('useAuthModal must be used within AuthModalProvider');
  });
});
