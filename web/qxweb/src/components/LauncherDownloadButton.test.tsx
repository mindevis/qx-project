import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/test-utils';
import { LauncherDownloadButton } from './LauncherDownloadButton';
import * as launcherDownload from '@/lib/launcherDownload';

describe('LauncherDownloadButton', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  it('opens download URL when configured', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(launcherDownload, 'resolveLauncherDownloadUrl').mockReturnValue(
      'https://releases.example/qx-launcher.exe',
    );
    const openSpy = vi.spyOn(launcherDownload, 'openLauncherDownload').mockImplementation(() => {});

    renderWithTheme(<LauncherDownloadButton type="primary" />);

    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(openSpy).toHaveBeenCalledWith('https://releases.example/qx-launcher.exe');
  });

  it('uses release info when provided', async () => {
    const user = userEvent.setup({ delay: null });
    const openSpy = vi.spyOn(launcherDownload, 'openLauncherDownload').mockImplementation(() => {});
    const resolveSpy = vi.spyOn(launcherDownload, 'resolveLauncherDownloadUrl');

    renderWithTheme(
      <LauncherDownloadButton
        release={{
          version: '2.0.0',
          download_url: '/downloads/qx-launcher.exe',
          filename: 'qx-launcher.exe',
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(resolveSpy).toHaveBeenCalled();
    expect(openSpy).toHaveBeenCalled();
  });
});
