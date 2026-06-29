import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
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

  it('shows build hint when URL is not configured', async () => {
    const user = userEvent.setup({ delay: null });
    const infoSpy = vi.spyOn(message, 'info');
    vi.spyOn(launcherDownload, 'resolveLauncherDownloadUrl').mockReturnValue(null);

    renderWithTheme(<LauncherDownloadButton />);

    await user.click(screen.getByRole('button', { name: /Скачать QXLauncher/ }));
    expect(infoSpy).toHaveBeenCalled();
    infoSpy.mockRestore();
  });
});
