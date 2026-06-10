import { type ReactNode } from 'react';
import { render } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { AuthModalProvider } from '@/auth/AuthModalContext';
import { ThemeProvider } from '@/theme/ThemeContext';

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
