import { afterEach, describe, expect, it, vi } from 'vitest';
import { openLauncherDownload, resolveLauncherDownloadUrl } from './launcherDownload';

describe('launcherDownload', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('returns configured download URL', () => {
    vi.stubEnv('VITE_LAUNCHER_DOWNLOAD_URL', '/downloads/qx-launcher.exe');
    expect(resolveLauncherDownloadUrl()).toBe('/downloads/qx-launcher.exe');
  });

  it('returns null when URL is not configured', () => {
    expect(resolveLauncherDownloadUrl()).toBeNull();
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
