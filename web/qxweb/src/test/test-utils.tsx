import { type ReactNode } from 'react';
import { expect } from 'vitest';
import { render, waitFor, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '@/auth/AuthContext';
import { AuthModalProvider } from '@/auth/AuthModalContext';
import { I18nProvider } from '@/i18n/I18nContext';
import { ThemeProvider } from '@/theme/ThemeContext';

export async function waitForNoDialog() {
  await waitFor(() => {
    expect(screen.queryAllByRole('dialog')).toHaveLength(0);
  });
}

export function I18nThemeWrapper({ children }: { children: ReactNode }) {
  return (
    <I18nProvider>
      <ThemeProvider>{children}</ThemeProvider>
    </I18nProvider>
  );
}

export function renderWithProviders(ui: ReactNode, route = '/') {
  return render(
    <I18nProvider>
      <ThemeProvider>
        <MemoryRouter initialEntries={[route]}>
          <AuthProvider>
            <AuthModalProvider>{ui}</AuthModalProvider>
          </AuthProvider>
        </MemoryRouter>
      </ThemeProvider>
    </I18nProvider>,
  );
}

export function renderWithTheme(ui: ReactNode) {
  return render(
    <I18nProvider>
      <ThemeProvider>{ui}</ThemeProvider>
    </I18nProvider>,
  );
}
