import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { ModSideBadge } from '@/components/ModSideBadge';
import { renderWithTheme } from '@/test/test-utils';

describe('ModSideBadge', () => {
  it('renders Russian client label', () => {
    renderWithTheme(
      <ModSideBadge item={{ client_side: 'required', server_side: 'unsupported' }} />,
    );
    expect(screen.getByText('Клиент')).toBeInTheDocument();
  });

  it('renders Russian both-sides label', () => {
    renderWithTheme(
      <ModSideBadge item={{ client_side: 'required', server_side: 'required' }} />,
    );
    expect(screen.getByText('Клиент + сервер')).toBeInTheDocument();
  });

  it('hides badge when side metadata is missing', () => {
    renderWithTheme(<ModSideBadge item={{}} />);
    expect(screen.queryByText('Клиент')).not.toBeInTheDocument();
    expect(screen.queryByText('Сервер')).not.toBeInTheDocument();
    expect(screen.queryByText('Клиент + сервер')).not.toBeInTheDocument();
  });
});
