import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BackendStatusProvider, useBackendStatus } from './BackendStatusContext';

vi.unmock('@/backend/BackendStatusContext');

vi.mock('@/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/client')>();
  return {
    ...actual,
    checkBackendHealth: vi.fn(),
  };
});

import { checkBackendHealth } from '@/api/client';

function StatusProbe() {
  const { available } = useBackendStatus();
  return <span data-testid="backend-status">{available ? 'up' : 'down'}</span>;
}

describe('BackendStatusProvider', () => {
  beforeEach(() => {
    vi.mocked(checkBackendHealth).mockReset();
  });

  it('marks backend unavailable when health check fails', async () => {
    vi.mocked(checkBackendHealth).mockResolvedValue(false);

    render(
      <BackendStatusProvider pollIntervalMs={50}>
        <StatusProbe />
      </BackendStatusProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('backend-status')).toHaveTextContent('down'));
  });

  it('recovers when health check succeeds on poll', async () => {
    vi.mocked(checkBackendHealth)
      .mockResolvedValueOnce(false)
      .mockResolvedValueOnce(true);

    render(
      <BackendStatusProvider pollIntervalMs={50}>
        <StatusProbe />
      </BackendStatusProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('backend-status')).toHaveTextContent('down'));
    await waitFor(() => expect(screen.getByTestId('backend-status')).toHaveTextContent('up'));
  });

  it('throws when hook is used outside provider', () => {
    expect(() => render(<StatusProbe />)).toThrow(
      'useBackendStatus must be used within BackendStatusProvider',
    );
  });
});
