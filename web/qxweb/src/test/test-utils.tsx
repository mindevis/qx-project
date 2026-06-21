import { type ReactNode } from 'react';
import { expect } from 'vitest';
import { render, waitFor, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { AuthModalProvider } from '@/auth/AuthModalContext';
import { ThemeProvider } from '@/theme/ThemeContext';

export async function waitForNoDialog() {
  await waitFor(() => {
    expect(screen.queryAllByRole('dialog')).toHaveLength(0);
  });
}

export function renderWithProviders(ui: ReactNode, route = '/') {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={[route]}>
        <AuthProvider>
          <AuthModalProvider>{ui}</AuthModalProvider>
        </AuthProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}
