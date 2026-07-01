export const DEFAULT_LAUNCHER_DOWNLOAD_PATH = '/downloads/qx-launcher.exe';

export type LauncherRelease = {
  version: string;
  download_url: string;
  filename: string;
};

export function resolveLauncherDownloadUrl(release?: Pick<LauncherRelease, 'download_url'> | null): string {
  const fromRelease = release?.download_url?.trim();
  if (fromRelease) return fromRelease;
  const configured = import.meta.env.VITE_LAUNCHER_DOWNLOAD_URL?.trim();
  if (configured) return configured;
  return DEFAULT_LAUNCHER_DOWNLOAD_PATH;
}

export function openLauncherDownload(url: string): void {
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.rel = 'noopener noreferrer';
  try {
    const resolved = new URL(url, window.location.href);
    if (resolved.origin === window.location.origin) {
      anchor.download = 'qx-launcher.exe';
    } else {
      anchor.target = '_blank';
    }
  } catch {
    anchor.target = '_blank';
  }
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
}
