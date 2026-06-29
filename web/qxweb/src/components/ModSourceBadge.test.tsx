import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/test-utils';
import { ModSourceBadge } from './ModSourceBadge';

describe('ModSourceBadge', () => {
  it('renders Modrinth source label', () => {
    renderWithTheme(<ModSourceBadge source="modrinth" />);
    expect(screen.getByText('Modrinth')).toBeInTheDocument();
  });

  it('renders CurseForge source label', () => {
    renderWithTheme(<ModSourceBadge source="curseforge" />);
    expect(screen.getByText('CurseForge')).toBeInTheDocument();
  });
});
