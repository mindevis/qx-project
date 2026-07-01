import { afterEach, describe, expect, it, vi } from 'vitest';
import { openLauncherDownload, resolveLauncherDownloadUrl } from './launcherDownload';

describe('launcherDownload', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('prefers release download URL', () => {
    expect(
      resolveLauncherDownloadUrl({
        version: '1.0.0',
        download_url: 'https://mc.qx-dev.ru/downloads/qx-launcher.exe',
        filename: 'qx-launcher.exe',
      }),
    ).toBe('https://mc.qx-dev.ru/downloads/qx-launcher.exe');
  });

  it('returns configured download URL', () => {
    vi.stubEnv('VITE_LAUNCHER_DOWNLOAD_URL', '/downloads/qx-launcher.exe');
    expect(resolveLauncherDownloadUrl()).toBe('/downloads/qx-launcher.exe');
  });

  it('falls back to default relative path', () => {
    expect(resolveLauncherDownloadUrl()).toBe('/downloads/qx-launcher.exe');
  });

  it('triggers same-origin download with filename', () => {
    const click = vi.fn();
    const remove = vi.fn();
    const anchor = {
      href: '',
      download: '',
      rel: '',
      target: '',
      click,
      remove,
    } as unknown as HTMLAnchorElement;
    const createElement = vi.spyOn(document, 'createElement').mockReturnValue(anchor);
    const appendChild = vi.spyOn(document.body, 'appendChild').mockImplementation(() => anchor);

    openLauncherDownload('/downloads/qx-launcher.exe');

    expect(anchor.download).toBe('qx-launcher.exe');
    expect(click).toHaveBeenCalled();
    expect(remove).toHaveBeenCalled();
    createElement.mockRestore();
    appendChild.mockRestore();
  });
});
