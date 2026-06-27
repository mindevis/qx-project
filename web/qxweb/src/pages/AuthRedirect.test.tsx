import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { Routes, Route } from 'react-router-dom';
import { renderWithProviders } from '@/test/test-utils';
import { AuthRedirect } from './AuthRedirect';
import { HomePage } from './HomePage';

describe('AuthRedirect', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('opens login modal and redirects home', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/auth/:mode" element={<AuthRedirect />} />
      </Routes>,
      '/auth/login',
    );

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
        'Единая экосистема для Minecraft',
      );
    });
  });

  it('opens register modal for register route', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/auth/:mode" element={<AuthRedirect />} />
      </Routes>,
      '/auth/register',
    );

    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Регистрация' })).toHaveAttribute('aria-selected', 'true'),
    );
  });

  it('honors returnTo query when opening auth route', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/launcher" element={<div>Launcher page</div>} />
        <Route path="/auth/:mode" element={<AuthRedirect />} />
      </Routes>,
      '/auth/login?returnTo=/launcher',
    );

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('Launcher page')).toBeInTheDocument();
    });
  });
});
