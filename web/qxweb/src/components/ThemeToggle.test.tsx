import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/test-utils';
import { ThemeToggle } from './ThemeToggle';

describe('ThemeToggle', () => {
  it('toggles theme', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<ThemeToggle />);

    expect(screen.getByRole('button', { name: 'Тёмная тема' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Тёмная тема' }));
    expect(screen.getByRole('button', { name: 'Светлая тема' })).toBeInTheDocument();
  });
});
