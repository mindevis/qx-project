import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/test-utils';
import { ThemeToggle } from './ThemeToggle';

describe('ThemeToggle', () => {
  it('switches between light and dark themes', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithProviders(<ThemeToggle />);

    const lightBtn = screen.getByRole('radio', { name: 'Светлая тема' });
    const darkBtn = screen.getByRole('radio', { name: 'Тёмная тема' });

    expect(lightBtn).toHaveAttribute('aria-checked', 'true');
    expect(darkBtn).toHaveAttribute('aria-checked', 'false');

    await user.click(darkBtn);

    expect(lightBtn).toHaveAttribute('aria-checked', 'false');
    expect(darkBtn).toHaveAttribute('aria-checked', 'true');
  });
});
