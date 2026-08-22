import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/test-utils';
import { DiscordInviteLink } from './DiscordInviteLink';
import { DISCORD_INVITE_URL } from '@/lib/community';

describe('DiscordInviteLink', () => {
  it('opens the Discord invite in a new tab', () => {
    renderWithProviders(<DiscordInviteLink />);

    const link = screen.getByRole('link', { name: 'Сообщество QXSystem в Discord' });
    expect(link).toHaveAttribute('href', DISCORD_INVITE_URL);
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    expect(link).toHaveTextContent('Discord');
  });
});
