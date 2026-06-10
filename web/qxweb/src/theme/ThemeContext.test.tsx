import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, useTheme } from './ThemeContext';
import { ThemeToggle } from '@/components/ThemeToggle';

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

    render(
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
    const user = userEvent.setup();

    render(
      <ThemeProvider>
        <ThemeProbe />
        <ThemeToggle />
      </ThemeProvider>,
    );

    expect(screen.getByTestId('theme-mode')).toHaveTextContent('light');
    expect(document.documentElement.dataset.theme).toBe('light');

    await user.click(screen.getByRole('button', { name: 'Тёмная тема' }));

    expect(screen.getByTestId('theme-mode')).toHaveTextContent('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(window.localStorage.getItem('qxweb-theme')).toBe('dark');
  });
});
