export const DEFAULT_LAUNCHER_DOWNLOAD_PATH = '/downloads/qx-launcher.exe';

export function resolveLauncherDownloadUrl(): string | null {
  const configured = import.meta.env.VITE_LAUNCHER_DOWNLOAD_URL?.trim();
  return configured || null;
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
