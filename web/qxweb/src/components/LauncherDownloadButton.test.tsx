import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { ThemeProvider } from '@/theme/ThemeContext';
import { LauncherDownloadButton } from './LauncherDownloadButton';

describe('LauncherDownloadButton', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  it('opens download URL when configured', async () => {
    const user = userEvent.setup({ delay: null });
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    vi.stubEnv('VITE_LAUNCHER_DOWNLOAD_URL', 'https://releases.example/qx-launcher.exe');

    render(
      <ThemeProvider>
        <LauncherDownloadButton type="primary" />
      </ThemeProvider>,
    );

    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(openSpy).toHaveBeenCalledWith(
      'https://releases.example/qx-launcher.exe',
      '_blank',
      'noopener,noreferrer',
    );
  });

  it('shows build hint when URL is not configured', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = vi.spyOn(message, 'info');

    render(
      <ThemeProvider>
        <LauncherDownloadButton />
      </ThemeProvider>,
    );

    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(infoSpy).toHaveBeenCalled();
    infoSpy.mockRestore();
  });
});
