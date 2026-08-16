import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { type ReactNode } from 'react';
import { ThemeProvider, useTheme } from './ThemeContext';
import { ThemeToggle } from '@/components/ThemeToggle';
import { I18nProvider } from '@/i18n/I18nContext';

function renderWithI18n(ui: ReactNode) {
  return render(<I18nProvider>{ui}</I18nProvider>);
}

function ThemeProbe() {
  const { mode } = useTheme();
  return <span data-testid="theme-mode">{mode}</span>;
}

describe('ThemeProvider', () => {
  beforeEach(() => {
    window.localStorage.clear();
    window.localStorage.setItem('qxweb-theme', 'light');
    document.documentElement.removeAttribute('data-theme');
  });

  afterEach(() => {
    window.localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('uses system preference when storage is empty', () => {
    window.localStorage.clear();
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('dark'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    renderWithI18n(
      <ThemeProvider>
        <ThemeProbe />
      </ThemeProvider>,
    );

    expect(screen.getByTestId('theme-mode')).toHaveTextContent('dark');
  });

  it('throws outside provider', () => {
    expect(() => render(<ThemeProbe />)).toThrow('useTheme must be used within ThemeProvider');
  });

  it('toggles theme and persists preference', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });

    renderWithI18n(
      <ThemeProvider>
        <ThemeProbe />
        <ThemeToggle />
      </ThemeProvider>,
    );

    expect(screen.getByTestId('theme-mode')).toHaveTextContent('light');
    expect(document.documentElement.dataset.theme).toBe('light');

    await user.click(screen.getByRole('radio', { name: 'Тёмная тема' }));

    expect(screen.getByTestId('theme-mode')).toHaveTextContent('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(window.localStorage.getItem('qxweb-theme')).toBe('dark');
  });

  it('toggleTheme switches dark back to light', async () => {
    window.localStorage.setItem('qxweb-theme', 'dark');
    const user = userEvent.setup();

    function ThemeToggler() {
      const { toggleTheme } = useTheme();
      return <button type="button" onClick={toggleTheme}>toggle</button>;
    }

    renderWithI18n(
      <ThemeProvider>
        <ThemeProbe />
        <ThemeToggler />
      </ThemeProvider>,
    );

    expect(screen.getByTestId('theme-mode')).toHaveTextContent('dark');
    await user.click(screen.getByRole('button', { name: 'toggle' }));
    expect(screen.getByTestId('theme-mode')).toHaveTextContent('light');
  });
});
